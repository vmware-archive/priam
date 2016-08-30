/*
Copyright (c) 2016 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package core

import (
	"errors"
	"fmt"
	"net/url"
	. "priam/util"
	"strconv"
)

// SCIM implementation of the users service
type SCIMUsersService struct{}

// SCIM implementation of the groups service
type SCIMGroupsService struct{}

// SCIM implementation of the roles service
type SCIMRolesService struct{}

const coreSchemaURN = "urn:scim:schemas:core:1.0"

// Define user information
type BasicUser struct {
	Name, Given, Family, Email, Pwd string `yaml:",omitempty,flow"`
}

type dispValue struct {
	Display, Value string `json:",omitempty"`
}

type nameAttr struct {
	GivenName, FamilyName string `json:",omitempty"`
}

type userAccount struct {
	Schemas               []string                                                   `json:",omitempty"`
	UserName              string                                                     `json:",omitempty"`
	Id                    string                                                     `json:",omitempty"`
	Active                bool                                                       `json:",omitempty"`
	Emails, Groups, Roles []dispValue                                                `json:",omitempty"`
	Meta                  *struct{ Created, LastModified, Location, Version string } `json:",omitempty"`
	Name                  *nameAttr                                                  `json:",omitempty"`
	WksExt                *struct{ InternalUserType, UserStatus string }             `json:"urn:scim:schemas:extension:workspace:1.0,omitempty"`
	Password              string                                                     `json:",omitempty"`
}

type memberValue struct {
	Value, Type, Operation string `json:",omitempty"`
}

type memberPatch struct {
	Schemas []string      `json:",omitempty"`
	Members []memberValue `json:",omitempty"`
}

// -- USERS
// @todo to put in scim_users.go

func (userService SCIMUsersService) DisplayEntity(ctx *HttpContext, username string) {
	scimGet(ctx, "Users", "userName", username)
}

func (userService SCIMUsersService) LoadEntities(ctx *HttpContext, fileName string) {
	var newUsers []BasicUser
	if err := GetYamlFile(fileName, &newUsers); err != nil {
		ctx.Log.Err("could not read file of bulk users: %v\n", err)
	} else {
		for k, v := range newUsers {
			if err := userService.AddEntity(ctx, &v); err != nil {
				ctx.Log.Err("Error adding user, line %d, name %s: %v\n", k+1, v.Name, err)
			} else {
				ctx.Log.Info("added user %s\n", v.Name)
			}
		}
	}
}

func (userService SCIMUsersService) AddEntity(ctx *HttpContext, entity interface{}) error {
	return scimAddUser(ctx, entity.(*BasicUser))
}

func (userService SCIMUsersService) UpdateEntity(ctx *HttpContext, name string, entity interface{}) {
	scimUpdateUser(ctx, name, entity.(*BasicUser))
}

func (userService SCIMUsersService) ListEntities(ctx *HttpContext, count int, filter string) {
	scimList(ctx, count, filter,
		"Users", "Users", "userName", "id", "emails",
		"display", "roles", "groups", "name",
		"givenName", "familyName", "value")
}

func (userService SCIMUsersService) UpdateMember(ctx *HttpContext, name, member string, remove bool) {
	ctx.Log.Err("Not implemented.")
}

func (userService SCIMUsersService) DeleteEntity(ctx *HttpContext, username string) {
	scimDelete(ctx, "Users", "userName", username)
}

// -- GROUPS
// @todo to put in scim_groups.go

func (groupService SCIMGroupsService) DisplayEntity(ctx *HttpContext, name string) {
	scimGet(ctx, "Groups", "displayName", name)
}

func (groupService SCIMGroupsService) LoadEntities(ctx *HttpContext, fileName string) {
	// not implemented
	ctx.Log.Err("Not implemented.")
}

func (groupService SCIMGroupsService) AddEntity(ctx *HttpContext, entity interface{}) error {
	// not implemented
	return errors.New("Not implemented")
}

func (groupService SCIMGroupsService) ListEntities(ctx *HttpContext, count int, filter string) {
	scimList(ctx, count, filter, "Groups", "displayName", "id", "members", "display")
}

func (groupService SCIMGroupsService) DeleteEntity(ctx *HttpContext, username string) {
	// not implemented
	ctx.Log.Err("Not implemented.")
}

func (groupService SCIMGroupsService) UpdateEntity(ctx *HttpContext, name string, entity interface{}) {
	// not implemented
	ctx.Log.Err("Not implemented.")
}

func (groupService SCIMGroupsService) UpdateMember(ctx *HttpContext, name, member string, remove bool) {
	scimMember(ctx, "Groups", "displayName", name, member, remove)
}

// -- ROLES
// @todo to put in scim_roles.go

func (roleService SCIMRolesService) DisplayEntity(ctx *HttpContext, name string) {
	scimGet(ctx, "Roles", "displayName", name)
}

func (roleService SCIMRolesService) LoadEntities(ctx *HttpContext, fileName string) {
	// not implemented
	ctx.Log.Err("Not implemented.")
}

func (roleService SCIMRolesService) AddEntity(ctx *HttpContext, entity interface{}) error {
	// not implemented
	return errors.New("Not implemented")
}

func (roleService SCIMRolesService) ListEntities(ctx *HttpContext, count int, filter string) {
	scimList(ctx, count, filter, "Roles", "displayName", "id")
}

func (roleService SCIMRolesService) DeleteEntity(ctx *HttpContext, username string) {
	// not implemented
	ctx.Log.Err("Not implemented.")
}

func (roleService SCIMRolesService) UpdateEntity(ctx *HttpContext, name string, entity interface{}) {
	// not implemented
	ctx.Log.Err("Not implemented.")
}

func (roleService SCIMRolesService) UpdateMember(ctx *HttpContext, name, member string, remove bool) {
	scimMember(ctx, "Roles", "displayName", name, member, remove)
}

// -- SCIM common code

func scimAddUser(ctx *HttpContext, u *BasicUser) error {
	acct := &userAccount{UserName: u.Name, Schemas: []string{coreSchemaURN}}
	acct.Password = u.Pwd
	acct.Name = &nameAttr{FamilyName: StringOrDefault(u.Family, u.Name), GivenName: StringOrDefault(u.Given, u.Name)}
	acct.Emails = []dispValue{{Value: StringOrDefault(u.Email, u.Name+"@example.com")}}
	ctx.Log.PP("add user: ", acct)
	return ctx.Request("POST", "scim/Users", acct, acct)
}

func scimUpdateUser(ctx *HttpContext, name string, u *BasicUser) {
	if id := scimNameToID(ctx, "Users", "userName", name); id != "" {
		acct := userAccount{UserName: u.Name, Schemas: []string{coreSchemaURN}}
		if u.Pwd != "" {
			acct.Password = u.Pwd
		}
		if u.Given != "" || u.Family != "" {
			acct.Name = &nameAttr{FamilyName: u.Family, GivenName: u.Given}
		}
		if u.Email != "" {
			acct.Emails = []dispValue{{Value: u.Email}}
		}

		if err := scimPatch(ctx, "Users", id, &acct); err != nil {
			ctx.Log.Err("Error updating user \"%s\": %v\n", name, err)
		} else {
			ctx.Log.Info("User \"%s\" updated\n", name)
		}
	}
}

func scimGetByName(ctx *HttpContext, resType, nameAttr, name string) (item map[string]interface{}, err error) {
	output := &struct {
		Resources                              []map[string]interface{}
		ItemsPerPage, TotalResults, StartIndex uint
		Schemas                                []string
	}{}
	vals := url.Values{"count": {"10000"}, "filter": {fmt.Sprintf("%s eq \"%s\"", nameAttr, name)}}
	path := fmt.Sprintf("scim/%v?%v", resType, vals.Encode())
	if err = ctx.Request("GET", path, nil, &output); err != nil {
		return
	}
	for _, v := range output.Resources {
		if CaselessEqual(name, v[nameAttr]) {
			if item != nil {
				return nil, fmt.Errorf("multiple %v found named \"%s\"", resType, name)
			} else {
				item = v
			}
		}
	}
	if item == nil {
		err = fmt.Errorf("no %v found named \"%s\"", resType, name)
	}
	return
}

func scimGetID(ctx *HttpContext, resType, nameAttr, name string) (string, error) {
	if item, err := scimGetByName(ctx, resType, nameAttr, name); err != nil {
		return "", err
	} else if id, ok := item["id"].(string); !ok {
		return "", fmt.Errorf("no id returned for \"%s\"", name)
	} else {
		return id, nil
	}
}

// @param count the number of records to return
// @param summaryLabels keys to filter the results of what to display
func scimList(ctx *HttpContext, count int, filter string, resType string, summaryLabels ...string) {
	vals := url.Values{}
	if count > 0 {
		vals.Set("count", strconv.Itoa(count))
	}
	if filter != "" {
		vals.Set("filter", filter)
	}
	path := fmt.Sprintf("scim/%s?%v", resType, vals.Encode())
	outp := make(map[string]interface{})
	if err := ctx.Request("GET", path, nil, &outp); err != nil {
		ctx.Log.Err("Error getting SCIM resources of type %s: %v\n", resType, err)
	} else {
		ctx.Log.PP(resType, outp["Resources"], summaryLabels...)
	}
}

func scimPatch(ctx *HttpContext, resType, id string, input interface{}) error {
	ctx.Header("X-HTTP-Method-Override", "PATCH")
	path := fmt.Sprintf("scim/%s/%s", resType, id)
	return ctx.Request("POST", path, input, nil)
}

func scimNameToID(ctx *HttpContext, resType, nameAttr, name string) string {
	if id, err := scimGetID(ctx, resType, nameAttr, name); err == nil {
		return id
	} else {
		ctx.Log.Err("Error getting SCIM %s ID of %s: %v\n", resType, name, err)
	}
	return ""
}

func scimMember(ctx *HttpContext, resType, nameAttr, rname, uname string, remove bool) {
	rid, uid := scimNameToID(ctx, resType, nameAttr, rname), scimNameToID(ctx, "Users", "userName", uname)
	if rid == "" || uid == "" {
		return
	}
	patch := memberPatch{Schemas: []string{coreSchemaURN}, Members: []memberValue{{Value: uid, Type: "User"}}}
	if remove {
		patch.Members[0].Operation = "delete"
	}
	if err := scimPatch(ctx, resType, rid, &patch); err != nil {
		ctx.Log.Err("Error updating SCIM resource %s of type %s: %v\n", rname, resType, err)
	} else {
		ctx.Log.Info("Updated SCIM resource %s of type %s\n", rname, resType)
	}
}

func scimGet(ctx *HttpContext, resType, nameAttr, rname string) {
	if item, err := scimGetByName(ctx, resType, nameAttr, rname); err != nil {
		ctx.Log.Err("Error getting SCIM resource named %s of type %s: %v\n", rname, resType, err)
	} else {
		ctx.Log.PP("", item)
	}
}

func scimDelete(ctx *HttpContext, resType, nameAttr, rname string) {
	if id := scimNameToID(ctx, resType, nameAttr, rname); id != "" {
		path := fmt.Sprintf("scim/%s/%s", resType, id)
		if err := ctx.Request("DELETE", path, nil, nil); err != nil {
			ctx.Log.Err("Error deleting %s %s: %v\n", resType, rname, err)
		} else {
			ctx.Log.Info("%s \"%s\" deleted\n", resType, rname)
		}
	}
}
