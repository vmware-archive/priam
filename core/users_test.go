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
	"fmt"
	"github.com/stretchr/testify/assert"
	. "github.com/vmware/priam/testaid"
	. "github.com/vmware/priam/util"
	"testing"
)

const (
	DEFAULT_USERNAME      = "john"
	DEFAULT_GROUP_NAME    = "saturday-night-fever"
	DEFAULT_ROLE_NAME     = "dancer"
	DEFAULT_GET_USER_URL  = "GET/scim/Users?count=10000&filter=userName+eq+%22" + DEFAULT_USERNAME + "%22"
	DEFAULT_POST_USER_URL = "POST/scim/Users/12345"
	DEFAULT_GET_GROUP_URL = "GET/scim/Groups?count=10000&filter=displayName+eq+%22" + DEFAULT_GROUP_NAME + "%22"
	YAML_USERS_FILE       = "../resources/newusers.yaml"
)

var aBasicUser = func() *BasicUser { return &BasicUser{Name: "john", Given: "travolta"} }

// Helpers to get useful handlers used in the tests
func scimDefaultUserHandler() func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		assert.Equal(t, "application/json", req.Accept)
		output := `{"Resources": [{ "userName" : "` + DEFAULT_USERNAME + `", "id": "12345"}]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}
}

func scimDefaultGroupHandler() func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		output := `{"Resources": [{ "displayName" : "` + DEFAULT_GROUP_NAME + `", "id": "6789"}]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}
}

func scimDefaultRoleHandler() func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		output := `{"Resources": [{ "displayName" : "` + DEFAULT_ROLE_NAME + `", "id": "123"}]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}
}

// Tests
func TestSetPassword(t *testing.T) {
	pwdH := func(t *testing.T, req *TstReq) *TstReply {
		assert.Contains(t, req.Input, "Password")
		return &TstReply{Status: 204}
	}
	srv := StartTstServer(t, map[string]TstHandler{
		"POST/scim/Users/12345": pwdH,
		DEFAULT_GET_USER_URL:    scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).UpdateEntity(ctx, "john", &BasicUser{Pwd: "travolta"})
	AssertOnlyInfoContains(t, ctx, `User "john" updated`)
}

func TestSetPasswordFailsIfScimPatchFails(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"POST/scim/Users/12345": ErrorHandler(404, "error set password"),
		DEFAULT_GET_USER_URL:    scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).UpdateEntity(ctx, "john", &BasicUser{Pwd: "travolta"})
	AssertErrorContains(t, ctx, "Error updating user \"john\": 404 Not Found\nerror set password")
}

func TestScimGetByNameWhenUserDoesNotExistReturnsError(t *testing.T) {
	errMessage := "test: john does not exist"
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, errMessage)})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	_, err := scimGetByName(ctx, "Users", "userName", "john")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), errMessage)
	}
}

func TestScimGetByNameWhenMultipleUsersReturnsError(t *testing.T) {
	multipleUsersHandler := func(t *testing.T, req *TstReq) *TstReply {
		output := `{"resources": [{ "userName" : "john", "id": "12345"}, { "userName" : "john", "id": "54321"}]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}

	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: multipleUsersHandler})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	_, err := scimGetByName(ctx, "Users", "userName", "john")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), "multiple Users found named \"john\"")
	}
}

func TestScimGetByNameWhenNoMatchReturnsError(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22patrick%22": scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	_, err := scimGetByName(ctx, "Users", "userName", "patrick")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), `no Users found named "patrick"`)
	}
}

func TestScimGetIdWithNoIdReturnsError(t *testing.T) {
	noIdHandler := func(t *testing.T, req *TstReq) *TstReply {
		output := `{"resources": [{ "userName" : "john" }]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: noIdHandler})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	_, err := scimGetID(ctx, "Users", "userName", "john")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), `no id returned for "john"`)
	}
}

func TestScimListWithCountAndFilter(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Users?count=3&filter=myfilter": scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).ListEntities(ctx, 3, "myfilter")
	AssertOnlyInfoContains(t, ctx, `id: "12345"`)
}

func TestScimListFilteredByLabel(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Users?filter=myfilter": scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimList(ctx, 0, "myfilter", "Users", "userName")
	AssertOnlyInfoContains(t, ctx, "userName: john")
	assert.NotContains(t, ctx.Log.InfoString(), "id")
}

func TestScimListWithNonExistingSummaryLabelsPrintsEmpty(t *testing.T) {
	// we're adding both URLs as older versions of go would normalize it and remove the trailing "?"
	// go 1.7.3 and + keep it
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Users?": scimDefaultUserHandler(),
		"GET/scim/Users":  scimDefaultUserHandler(),
	})
	defer srv.Close()

	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimList(ctx, 0, "", "Users", "IDontExist")
	AssertOnlyInfoContains(t, ctx, "")
}

func TestScimListReturnsErrorOnInvalidRequest(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Users?filter=myfilter": ErrorHandler(404, "error scim list")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).ListEntities(ctx, 0, "myfilter")
	AssertErrorContains(t, ctx, "Error getting SCIM resources of type Users: 404 Not Found\nerror scim list\n")
}

func TestScimGetNameExists(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).DisplayEntity(ctx, "john")
	AssertOnlyInfoContains(t, ctx, "userName: john")
}

func TestScimGetWhenNameDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "error scim get")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).DisplayEntity(ctx, "john")
	AssertErrorContains(t, ctx, "Error getting SCIM resource named john of type Users: 404 Not Found\nerror scim get\n")
}

func TestScimAddUser(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST/scim/Users": scimDefaultUserHandler()})
	defer srv.Close()
	new(SCIMUsersService).AddEntity(ctx, aBasicUser())
	AssertOnlyInfoContains(t, ctx, "username: john")
	AssertOnlyInfoContains(t, ctx, "---- add user:  ----")
}

func TestScimAddUserReturnsErrorOnScimError(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST/scim/Users": ErrorHandler(404, "error scim add")})
	defer srv.Close()
	new(SCIMUsersService).AddEntity(ctx, aBasicUser())
	AssertErrorContains(t, ctx, "404 Not Found\nerror scim add\n")
}

func TestScimUpdateUserFailedIfUserDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "not found")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).UpdateEntity(ctx, "john", &BasicUser{Name: "john", Given: "wayne"})
	AssertErrorContains(t, ctx, "Error getting SCIM Users ID of john: 404 Not Found")
}

func TestScimUpdateUserFailedIfPatchCommandFails(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler(),
		// response does not matter, only body could be tested
		"POST/scim/Users/12345": ErrorHandler(404, "error scim patch")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).UpdateEntity(ctx, "john", &BasicUser{Name: "john", Given: "johnny"})
	AssertErrorContains(t, ctx, `Error updating user "john": 404 Not Found`)
}

// Helper for testing SCIM update
func scimUpdateDefaultUserWith(t *testing.T, name string, updatedUser *BasicUser) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler(),
		// response does not matter, only body could be tested
		"POST/scim/Users/12345": scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).UpdateEntity(ctx, name, updatedUser)
	AssertOnlyInfoContains(t, ctx, fmt.Sprintf("User \"%s\" updated", name))
}

func TestScimUpdateUserName(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &BasicUser{Name: "newjohn", Given: "johnny"})
}

func TestScimUpdateUserGivenName(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &BasicUser{Name: "john", Given: "johnny"})
}

func TestScimUpdateUserFamilyName(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &BasicUser{Name: "john", Family: "wayne"})
}

func TestScimUpdateUserEmail(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &BasicUser{Name: "john", Email: "j@travolta.com"})
}

func TestScimDeleteFailsIfUserDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "test error")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimDelete(ctx, "Users", "userName", "john")
	AssertErrorContains(t, ctx, "Error getting SCIM Users ID of john: 404 Not Found")
}

func TestScimDeleteFailsIfDeleteCommandFails(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL:      scimDefaultUserHandler(),
		"DELETE/scim/Users/12345": ErrorHandler(404, "error scim delete")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimDelete(ctx, "Users", "userName", "john")
	AssertErrorContains(t, ctx, "Error deleting Users john: 404 Not Found\nerror scim delete")
}

func TestScimDelete(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler(),
		// response does not matter as long as this is 20x
		"DELETE/scim/Users/12345": scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).DeleteEntity(ctx, "john")
	AssertOnlyInfoContains(t, ctx, `Users "john" deleted`)
}

func TestScimMemberReturnsWhenNoResourceId(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "error scim members")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimMember(ctx, "Users", "userName", "john", "john", false)
	AssertErrorContains(t, ctx, "Error getting SCIM Users ID of john")
}

func TestAddScimMember(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL:  scimDefaultUserHandler(),
		DEFAULT_POST_USER_URL: scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimMember(ctx, "Users", "userName", "john", "john", false)
	AssertOnlyInfoContains(t, ctx, "Updated SCIM resource john of type Users\n")
}

func TestRemoveScimMember(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL:  scimDefaultUserHandler(),
		DEFAULT_POST_USER_URL: scimDefaultUserHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimMember(ctx, "Users", "userName", "john", "john", true)
	AssertOnlyInfoContains(t, ctx, "Updated SCIM resource john of type Users\n")
}

func TestRemoveScimMemberReturnsErrorIfScimPatchFailed(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_USER_URL:  scimDefaultUserHandler(),
		DEFAULT_POST_USER_URL: ErrorHandler(404, "error scim patch members")})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	scimMember(ctx, "Users", "userName", "john", "john", true)
	AssertErrorContains(t, ctx, "Error updating SCIM resource john of type Users")
}

func TestLoadUsersFromYaml(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST/scim/Users": scimDefaultUserHandler()})
	defer srv.Close()
	new(SCIMUsersService).LoadEntities(ctx, YAML_USERS_FILE)
	AssertOnlyInfoContains(t, ctx, "User 'joe1' successfully added")
}

func TestLoadUsersFromYamlFailedIfAddUserFailed(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST/scim/Users": ErrorHandler(404, "error scim add user")})
	defer srv.Close()
	new(SCIMUsersService).LoadEntities(ctx, YAML_USERS_FILE)
	AssertErrorContains(t, ctx, "Error creating user 'joe1': 404 Not Found")
}

func TestLoadUsersFromYamlFailedIfYamlFileDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMUsersService).LoadEntities(ctx, "newusers-does-not-exist.yaml")
	AssertErrorContains(t, ctx, "could not read file of bulk users")
}

// Tests for GROUPS
// @todo To be put in groups_test.go?

func TestGetGroup(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		DEFAULT_GET_GROUP_URL: scimDefaultGroupHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMGroupsService).DisplayEntity(ctx, DEFAULT_GROUP_NAME)
	AssertOnlyInfoContains(t, ctx, "displayName: "+DEFAULT_GROUP_NAME)
}

func TestListGroups(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Groups?count=3&filter=myfilter": scimDefaultGroupHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMGroupsService).ListEntities(ctx, 3, "myfilter")
	AssertOnlyInfoContains(t, ctx, `id: "6789"`)
	AssertOnlyInfoContains(t, ctx, "displayName: "+DEFAULT_GROUP_NAME)
}

// Tests for ROLES
// @todo To be put in roles_test.go?

func TestListRoles(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{
		"GET/scim/Roles?count=3&filter=myfilter": scimDefaultRoleHandler()})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	new(SCIMRolesService).ListEntities(ctx, 3, "myfilter")
	AssertOnlyInfoContains(t, ctx, `id: "123"`)
	AssertOnlyInfoContains(t, ctx, "displayName: "+DEFAULT_ROLE_NAME)
}
