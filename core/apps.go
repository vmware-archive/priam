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
	"github.com/pborman/uuid"
	. "github.com/vmware/priam/util"
	"strings"
)

// this type is only used for testing. the test code implements the
// json.Marshaler interface so that it can return an error
type jsonMarshalTester string

// the IDM application service
type IDMApplicationService struct {
}

type priamApp struct {
	Name                  string                   `json:"name,omitempty" yaml:"name,omitempty"`
	Uuid                  string                   `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	PackageVersion        string                   `json:"packageVersion,omitempty" yaml:"packageVersion,omitempty"`
	Description           string                   `json:"description,omitempty" yaml:"description,omitempty"`
	IconFile              string                   `json:"iconFile,omitempty" yaml:"iconFile,omitempty"`
	EntitleGroup          string                   `json:"entitleGroup,omitempty" yaml:"entitleGroup,omitempty"`
	EntitleUser           string                   `json:"entitleUser,omitempty" yaml:"entitleUser,omitempty"`
	ResourceConfiguration map[string]interface{}   `json:"resourceConfiguration" yaml:"resourceConfiguration,omitempty"`
	AccessPolicy          string                   `json:"accessPolicy,omitempty" yaml:"accessPolicy,omitempty"`
	AccessPolicySetUuid   string                   `json:"accessPolicySetUuid,omitempty" yaml:"accessPolicySetUuid,omitempty"`
	CatalogItemType       string                   `json:"catalogItemType,omitempty" yaml:"catalogItemType,omitempty"`
	JsonTester            jsonMarshalTester        `json:"jsonTester,omitempty" yaml:"jsonTester,omitempty"`
	Labels                []map[string]interface{} `json:"labels,omitempty" yaml:"labels,omitempty"`
	AuthInfo              map[string]interface{}   `json:"authInfo,omitempty" yaml:"authInfo,omitempty"`
}

type manifestApp struct {
	Name, Memory, Path, BuildPack string
	Instances                     int
	Env                           map[string]string
	Workspace                     priamApp
}

type itemResponse struct {
	Links map[string]interface{}   `json:"_links,omitempty" yaml:"_links,omitempty"`
	Items []map[string]interface{} `json:",omitempty" yaml:",omitempty"`
}

// Display application info
func (service IDMApplicationService) Display(ctx *HttpContext, appName string) {
	appGet(ctx, appName)
}

// Delete given application from the catalog
func (service IDMApplicationService) Delete(ctx *HttpContext, appName string) {
	appDelete(ctx, appName)
}

// List all applications in the catalog
func (service IDMApplicationService) List(ctx *HttpContext, count int, filter string) {
	appList(ctx, count, filter)
}

// Publish an application
func (service IDMApplicationService) Publish(ctx *HttpContext, manifestFile string) {
	PublishApps(ctx, manifestFile)
}

func accessPolicyId(ctx *HttpContext, name string) string {
	outp := &itemResponse{}
	ctx.Accept("accesspolicyset.list")
	if err := ctx.Request("GET", "accessPolicies", nil, &outp); err != nil {
		ctx.Log.Err("Error getting access policies: %v\n", err)
		return ""
	}
	for _, item := range outp.Items {
		if name == "" && item["base"] == true || CaselessEqual(name, item["name"]) {
			if s, ok := item["uuid"].(string); ok {
				return s
			}
		}
	}
	ctx.Log.Err("Could not find access policy uuid\n")
	return ""
}

// input name, uuid
// output uuid of existing app with the input uuid or uuid of first app with name
func checkAppExists(ctx *HttpContext, name, uuid string) (outid string, err error) {
	outp := &itemResponse{}
	ctx.Accept("catalog.summary.list").ContentType("catalog.search")
	if err = ctx.Request("POST", "catalogitems/search?pageSize=10000", "{}", &outp); err == nil {
		for _, item := range outp.Items {
			if CaseEqual(uuid, item["uuid"]) {
				outid = uuid
				return
			}
			if CaselessEqual(name, item["name"]) && outid == "" {
				outid = InterfaceToString(item["uuid"])
			}
		}
	}
	return
}

func getAppUuid(ctx *HttpContext, name string) (uuid, mtype string, err error) {
	inp, outp := fmt.Sprintf(`{"nameFilter":"%s"}`, EscapeQuotes(name)), new(itemResponse)
	ctx.Accept("catalog.summary.list").ContentType("catalog.search")
	if err = ctx.Request("POST", "catalogitems/search?pageSize=10000", &inp, &outp); err == nil {
		for _, item := range outp.Items {
			if u, ok := item["uuid"].(string); ok && CaselessEqual(name, item["name"]) {
				if uuid != "" {
					err = fmt.Errorf("Multiple apps with name \"%s\"", name)
					return
				}
				uuid = u
				if mt, ok := item["catalogItemType"].(string); ok {
					mtype = "catalog." + strings.ToLower(mt)
				}
			}
		}
	}
	if uuid == "" {
		err = fmt.Errorf("No app found with name \"%s\"", name)
	}
	return
}

func getAppByUuid(ctx *HttpContext, uuid, mtype string) (app map[string]interface{}, err error) {
	app = make(map[string]interface{})
	err = ctx.Accept(mtype).Request("GET", fmt.Sprintf("catalogitems/%s", uuid), nil, &app)
	return
}

func PublishApps(ctx *HttpContext, manifile string) {
	if manifile == "" {
		manifile = "manifest.yaml"
	}
	var manifest struct{ Applications []manifestApp }
	if err := GetYamlFile(manifile, &manifest); err != nil {
		ctx.Log.Err("Error getting manifest: %v\n", err)
		return
	}
	for _, v := range manifest.Applications {
		var w = &v.Workspace
		if w.Name == "" {
			w.Name = v.Name
		}
		if w.AccessPolicySetUuid == "" {
			if w.AccessPolicySetUuid = accessPolicyId(ctx, w.AccessPolicy); w.AccessPolicySetUuid == "" {
				ctx.Log.Err("Skipping app %s\n", w.Name) // accessPolicyID logs any errors so user knows reason for skip
				continue
			}
			w.AccessPolicy = ""
		} else if w.AccessPolicy != "" {
			ctx.Log.Err("Invalid manifest for %s: both accessPolicy \"%s\" and AccessPolicySetUuid \"%s\" cannot be specified\n",
				w.Name, w.AccessPolicy, w.AccessPolicySetUuid)
			continue
		}
		method, path, errVerb, successVerb := "POST", "catalogitems", "adding", "added"
		id, err := checkAppExists(ctx, w.Name, w.Uuid)
		if err != nil {
			ctx.Log.Err("Error checking if app %s exists: %v\n", w.Name, err)
			continue
		}
		if id != "" {
			method, errVerb, successVerb, w.Uuid = "PUT", "updating", "updated", id
			path += "/" + id
		}
		if w.Uuid == "" {
			w.Uuid = uuid.New()
		}
		mtype := "catalog." + strings.ToLower(w.CatalogItemType)
		iconFile, entitleGrp, entitleUser := w.IconFile, w.EntitleGroup, w.EntitleUser
		w.IconFile, w.EntitleGroup, w.EntitleUser = "", "", ""
		content, err := ToJson(w)
		if err != nil {
			ctx.Log.Err("Error converting app %s to JSON: %v\n", w.Name, err)
			continue
		}
		if iconFile == "" {
			err = ctx.Accept(mtype).ContentType(mtype).Request(method, path, content, nil)
		} else {
			err = ctx.FileUploadRequest(method, path, "catalogitem", mtype, content, iconFile, nil)
		}
		if err != nil {
			ctx.Log.Err("Error %s %s to the catalog: %v\n", errVerb, w.Name, err)
			continue
		}
		ctx.Log.Info("App \"%s\" %s to the catalog\n", w.Name, successVerb)
		maybeEntitle(ctx, w.Uuid, entitleGrp, "group", "displayName", w.Name)
		maybeEntitle(ctx, w.Uuid, entitleUser, "user", "userName", w.Name)
	}
}

func appDelete(ctx *HttpContext, name string) {
	if uuid, _, err := getAppUuid(ctx, name); err != nil {
		ctx.Log.Err("Error getting app info by name: %v\n", err)
	} else if err := ctx.Request("DELETE", fmt.Sprintf("catalogitems/%s", uuid), nil, nil); err != nil {
		ctx.Log.Err("Error deleting app %s from catalog: %v\n", name, err)
	} else {
		ctx.Log.Info("app %s deleted\n", name)
	}
}

func appGet(ctx *HttpContext, name string) {
	if uuid, mtype, err := getAppUuid(ctx, name); err != nil {
		ctx.Log.Err("Error getting app info by name: %v\n", err)
	} else if app, err := getAppByUuid(ctx, uuid, mtype); err != nil {
		ctx.Log.Err("Error getting app info by uuid: %v\n", err)
	} else {
		ctx.Log.PP("App "+name, app)
	}
}

func appList(ctx *HttpContext, count int, filter string) {
	if count == 0 {
		count = 10000
	}
	path, input := fmt.Sprintf("catalogitems/search?pageSize=%v", count), "{}"
	if filter != "" {
		input = fmt.Sprintf(`{"nameFilter":"%s"}`, EscapeQuotes(filter))
	}
	body := make(map[string]interface{})
	ctx.Accept("catalog.summary.list").ContentType("catalog.search")
	if err := ctx.Request("POST", path, input, &body); err != nil {
		ctx.Log.Err("Error: %v\n", err)
	} else {
		ctx.Log.PP("Apps", body["items"], "name", "description", "catalogItemType", "uuid")
	}
}
