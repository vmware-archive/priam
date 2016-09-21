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

// The application service interface.
type AppTemplateService interface {
	// Add creates a new app template
	Add(ctx *HttpContext, name string, templateInfo map[string]interface{})

	// Get displays the given application template by name
	Get(ctx *HttpContext, name string)

	// Delete deletes the given application template by name
	Delete(ctx *HttpContext, name string)

	// List displays all application templates
	List(ctx *HttpContext)
}

// the IDM application template service
type IDMAppTemplateService struct{}

const templatePath = "oauth2apptemplates"

var templateSummaryFields = []string{"items", "appProductId", "accessTokenTTL", "authGrantTypes",
	"redirectUri", "displayUserGrant", "length", "refreshTokenTTL", "resourceUuid", "scope", "tokenType"}

// Get oauth2 application template info
func (service IDMAppTemplateService) Get(ctx *HttpContext, name string) {
	ctx.GetPrintJson("Get App Template "+name, templatePath+"/"+name, "oauth2apptemplate", templateSummaryFields...)
}

// Delete oauth2 application template
func (service IDMAppTemplateService) Delete(ctx *HttpContext, name string) {
	if err := ctx.Request("DELETE", templatePath+"/"+name, nil, nil); err != nil {
		ctx.Log.Err("Error deleting template \"%s\": %v\n", name, err)
	} else {
		ctx.Log.Info("Template \"%s\" deleted\n", name)
	}
}

// List all oauth2 application templates
func (service IDMAppTemplateService) List(ctx *HttpContext) {
	ctx.GetPrintJson("List App Templates", templatePath, "oauth2apptemplate.list", templateSummaryFields...)
}

// add a new oauth2 application template
func (service IDMAppTemplateService) Add(ctx *HttpContext, name string, templateInfo map[string]interface{}) {
	if err := ctx.ContentType("oauth2apptemplate").Request("POST", templatePath, templateInfo, nil); err != nil {
		ctx.Log.Err("Error adding template \"%s\": %v\n", name, err)
	} else {
		ctx.Log.Info("Successfully added template \"%s\"\n", name)
	}
}
