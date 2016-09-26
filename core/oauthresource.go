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
	. "github.com/vmware/priam/util"
)

// The oauth reseource service interface.
type OauthResource interface {
	// Add creates a new oauth resource
	Add(ctx *HttpContext, name string, info map[string]interface{})

	// Get displays the given oauth resource by name
	Get(ctx *HttpContext, name string)

	// Delete removes an oauth resource by name
	Delete(ctx *HttpContext, name string)

	// List displays all oauth resources of a type
	List(ctx *HttpContext)
}

// The generic resource service interface.
type OauthResourceService struct {
	resType, path, itemMT, listMT string
	summaryFields                 []string
}

var AppTemplateService = &OauthResourceService{"App Template", "oauth2apptemplates", "oauth2apptemplate",
	"oauth2apptemplate.list", []string{"items", "appProductId", "accessTokenTTL", "authGrantTypes",
		"redirectUri", "displayUserGrant", "length", "refreshTokenTTL", "resourceUuid", "scope", "tokenType"}}

var OauthClientService = &OauthResourceService{"Oauth2 Client", "oauth2clients", "oauth2client",
	"oauth2clientsummarylist", []string{"items", "refreshTokenTTL", "accessTokenTTL", "strData",
		"resourceUuid", "tokenLength", "displayUserGrant", "authGrantTypes", "internalSystemClient",
		"redirectUri", "clientId", "rememberAs", "scope", "tokenType", "inheritanceAllowed", "secret",
	}}

// Get displays oauth2 resource info
func (rs *OauthResourceService) Get(ctx *HttpContext, name string) {
	ctx.GetPrintJson("Get "+rs.resType+" "+name, rs.path+"/"+name, rs.itemMT, rs.summaryFields...)
}

// Delete removes an oauth2 resource
func (rs *OauthResourceService) Delete(ctx *HttpContext, name string) {
	if err := ctx.ContentType(rs.itemMT).Accept(rs.itemMT).Request("DELETE", rs.path+"/"+name, nil, nil); err != nil {
		ctx.Log.Err("Error deleting %s \"%s\": %v\n", rs.resType, name, err)
	} else {
		ctx.Log.Info("%s \"%s\" deleted\n", rs.resType, name)
	}
}

// List all oauth2 resources of a type
func (rs *OauthResourceService) List(ctx *HttpContext) {
	ctx.GetPrintJson("List "+rs.resType+"s", rs.path, rs.listMT, rs.summaryFields...)
}

// add a new oauth2 resource
func (rs *OauthResourceService) Add(ctx *HttpContext, name string, info map[string]interface{}) {
	if err := ctx.ContentType(rs.itemMT).Request("POST", rs.path, info, nil); err != nil {
		ctx.Log.Err("Error adding %s \"%s\": %v\n", rs.resType, name, err)
	} else {
		ctx.Log.Info("Successfully added %s \"%s\"\n", rs.resType, name)
	}
}
