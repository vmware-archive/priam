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
	. "priam/testaid"
	"strings"
	"testing"
)

func TestGetEntitlementForUser(t *testing.T) {
	checkGetEntitlementReturns(t, "user", "Users", "testid67")
}

func TestGetEntitlementForGroup(t *testing.T) {
	checkGetEntitlementReturns(t, "group", "Groups", "testid67")
}

func TestGetEntitlementForApp(t *testing.T) {
	checkGetEntitlementReturns(t, "app", "catalogitems", "foo")
}

func TestGetEntitlementForUnknownScimUser(t *testing.T) {
	errorReply := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Status: 404, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22foo%22": errorReply}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	GetEntitlement(ctx, "user", "foo")
	AssertErrorContains(t, ctx, "Error getting SCIM Users ID of foo: 404 Not Found")
}

func TestGetEntitlementForUnknownUserEntitlement(t *testing.T) {
	entErrorReply := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Status: 404, StatusMsg: "test: foo does not exist"}
	}
	idH := func(t *testing.T, req *TstReq) *TstReply {
		output := fmt.Sprintf(`{"Resources": [{ "userName" : "foo", "displayName" : "foo", "id": "%s"}]}`, "test-fail")
		return &TstReply{Output: output, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22foo%22": idH,
		"GET/entitlements/definitions/users/test-fail":            entErrorReply}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	GetEntitlement(ctx, "user", "foo")
	AssertErrorContains(t, ctx, "Error: 404 Not Found")
	AssertErrorContains(t, ctx, "test: foo does not exist")
}

func TestCreateEntitlementForUser(t *testing.T) {
	entReply := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{"test": "unused"}`, ContentType: "application/json"}
	}
	idH := func(t *testing.T, req *TstReq) *TstReply {
		output := `{"resources": [{ "userName" : "patrick", "id": "12345"}]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22patrick%22": idH,
		"POST/entitlements/definitions":                               entReply}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	maybeEntitle(ctx, "baby", "patrick", "user", "userName", "dance")
	AssertOnlyInfoContains(t, ctx, `Entitled user "patrick" to app "dance"`)
}

// Test user.
// @todo test group as well.
func TestCreateEntitlementFailedForUnknownUser(t *testing.T) {
	errorReply := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Status: 404, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22patrick%22": errorReply}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	maybeEntitle(ctx, "baby", "patrick", "user", "userName", "dance")
	AssertErrorContains(t, ctx, `Could not entitle user "patrick" to app "dance", error: 404 Not Found`)
}

// common method to test getting basic entitlements
func checkGetEntitlementReturns(t *testing.T, entity, rType, rID string) {
	entH := func(t *testing.T, req *TstReq) *TstReply {
		output := `{"items": [{ "Entitlements" : "bar"}]}`
		return &TstReply{Output: output, ContentType: "application/json"}
	}
	idH := func(t *testing.T, req *TstReq) *TstReply {
		output := fmt.Sprintf(`{"resources": [{ "userName" : "foo", "displayName" : "foo", "id": "%s"}]}`, rID)
		return &TstReply{Output: output, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22foo%22":                 idH,
		"GET/scim/Groups?count=10000&filter=displayName+eq+%22foo%22":             idH,
		"GET/" + "entitlements/definitions/" + strings.ToLower(rType) + "/" + rID: entH}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	GetEntitlement(ctx, entity, "foo")
	AssertOnlyInfoContains(t, ctx, "Entitlements: bar")
}
