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
	"github.com/stretchr/testify/assert"
	. "github.com/vmware/priam/testaid"
	"testing"
)

const appTemplateMT = "oauth2apptemplate+json"
const appTemplateListMT = "oauth2apptemplate.list+json"
const appTemplateItem = `{"appProductId": "olaf", "resourceUuid": "6c48beb6-afb1-44bc-ad7f-980214ee346c"}`
const appTemplateList = `{"items": [` + appTemplateItem + `]}`
const appTemplateBasePath = "/oauth2apptemplates"

func TestAppTemplateGet(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + appTemplateBasePath + "/olaf": func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, appTemplateMT, req.Accept)
			return &TstReply{Status: 200, Output: appTemplateItem, ContentType: appTemplateMT}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).Get(ctx, "olaf")
	AssertOnlyInfoContains(t, ctx, `appProductId: olaf`)
}

func TestAppTemplateNotFound(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + appTemplateBasePath + "/hans": func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, appTemplateMT, req.Accept)
			return &TstReply{Status: 404, Output: `{"message":"oauth2apptemplate.not.found"}`,
				ContentType: "application/json"}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).Get(ctx, "hans")
	AssertOnlyErrorContains(t, ctx, `Error: 404 Not Found`)
	AssertOnlyErrorContains(t, ctx, `oauth2apptemplate.not.found`)
}

func TestAppTemplateList(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + appTemplateBasePath: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, appTemplateListMT, req.Accept)
			return &TstReply{Status: 200, Output: appTemplateList, ContentType: appTemplateListMT}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).List(ctx)
	AssertOnlyInfoContains(t, ctx, `items`)
	AssertOnlyInfoContains(t, ctx, `appProductId: olaf`)
}

func TestAppTemplateListRejected(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + appTemplateBasePath: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, appTemplateListMT, req.Accept)
			return &TstReply{Status: 403}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).List(ctx)
	AssertOnlyErrorContains(t, ctx, `Error: 403 Forbidden`)
}

func TestAppTemplateDelete(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"DELETE" + appTemplateBasePath + "/olaf": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Status: 200}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).Delete(ctx, "olaf")
	AssertOnlyInfoContains(t, ctx, `Template "olaf" deleted`)
}

func TestAppTemplateDeleteRejected(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"DELETE" + appTemplateBasePath + "/olaf": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Status: 500, Output: "elsa's in a bad mood"}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).Delete(ctx, "olaf")
	AssertOnlyErrorContains(t, ctx, `Error deleting template "olaf": 500 Internal Server Error`)
}

func TestAppTemplateAdd(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"POST" + appTemplateBasePath: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, appTemplateMT, req.ContentType)
			assert.Equal(t, `{"appProductId":"olaf"}`, req.Input)
			return &TstReply{Status: 200, Output: appTemplateItem, ContentType: appTemplateMT}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).Add(ctx, "olaf", map[string]interface{}{"appProductId": "olaf"})
	AssertOnlyInfoContains(t, ctx, `Successfully added template "olaf"`)
}

func TestAppTemplateRejected(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"POST" + appTemplateBasePath: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, appTemplateMT, req.ContentType)
			return &TstReply{Status: 406, ContentType: "application/json",
				Output: `{"message":"clumsy attempt at kindness is scorned by elsa"}`}
		}})
	defer srv.Close()
	new(IDMAppTemplateService).Add(ctx, "sven", map[string]interface{}{"appProductId": "sven",
		"scope": "coffee", "redirectUri": "elsa:://insecure"})
	AssertOnlyErrorContains(t, ctx, `clumsy`)
	AssertOnlyErrorContains(t, ctx, `406`)
}
