package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"fmt"
	"strings"
)

func TestGetEntitlementWithNoArgsShowsHelp(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "entitlement"); ctx != nil {
		assert.Contains(t, ctx.info, "USAGE")
	}
}

func TestGetEntitlementWithNoTypeShowsError(t *testing.T) {
	if ctx := runner(t, newTstCtx(" "), "entitlement", "get"); ctx != nil {
		assert.Contains(t, ctx.err, "at least 2 arguments must be given")
	}
}

func TestGetEntitlementWithNoNameShowsError(t *testing.T) {
	types := [...]string{"user", "app", "group"}
	for i := range types {
		if ctx := runner(t, newTstCtx(" "), "entitlement", "get", types[i]); ctx != nil {
			assert.Contains(t, ctx.err, "at least 2 arguments must be given")
		}
	}
}

func TestGetEntitlementWithWrongTypeShowsError(t *testing.T) {
	if ctx := runner(t, newTstCtx(" "), "entitlement", "get", "actor", "swayze"); ctx != nil {
		assert.Contains(t, ctx.err, "First parameter of 'get' must be user, group or app")
	}
}

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
	errorReply := func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{status: 404, contentType: "application/json"}
	}
	paths := map[string]tstHandler{
		"POST" + vidmTokenPath:                 tstClientCredGrant,
		"GET" + vidmBasePath + "scim/Users":     errorReply}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "entitlement", "get", "user", "foo"); ctx != nil {
		assert.Contains(t, ctx.err, "Error getting SCIM Users ID of foo: 404 Not Found")
	}
}

func TestGetEntitlementForUnknownUserEntitlement(t *testing.T) {
	entErrorReply := func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{status: 404, statusMsg: "test: foo does not exist"}
	}
	idH := func(t *testing.T, req *tstReq) *tstReply {
		output := fmt.Sprintf(`{"resources": [{ "userName" : "foo", "displayName" : "foo", "id": "%s"}]}`, "test-fail")
		return &tstReply{output: output, contentType: "application/json"}
	}
	paths := map[string]tstHandler{
		"POST" + vidmTokenPath:                 tstClientCredGrant,
		"GET" + vidmBasePath + "scim/Users":     idH,
		"GET" + vidmBasePath + "entitlements/definitions/users/test-fail":  entErrorReply}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "entitlement", "get", "user", "foo"); ctx != nil {
		assert.Contains(t, ctx.err, "Error: 404 Not Found")
		assert.Contains(t, ctx.err, "test: foo does not exist")
		assert.Empty(t, ctx.info)
	}
}

func TestCreateEntitlementForUser(t *testing.T) {
	entReply := func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{output: `{"test": "unused"}`, contentType: "application/json"}
	}
	idH := func(t *testing.T, req *tstReq) *tstReply {
		output := `{"resources": [{ "userName" : "patrick", "id": "12345"}]}`
		return &tstReply{output: output, contentType: "application/json"}
	}
	paths := map[string]tstHandler{
		"GET/scim/Users" : idH,
		"POST/entitlements/definitions":  entReply}
	srv := StartTstServer(t, paths)
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	maybeEntitle(ctx, "baby", "patrick", "user", "userName", "dance")
	assert.Empty(t, ctx.log.errString())
	assert.Contains(t, ctx.log.infoString(), `Entitled user "patrick" to app "dance"`)
}

// Test user.
// @todo test group.
func TestCreateEntitlementFailedForUnknownUser(t *testing.T) {
	errorReply := func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{status: 404, contentType: "application/json"}
	}
	paths := map[string]tstHandler{
		"GET/scim/Users" : errorReply}
	srv := StartTstServer(t, paths)
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	maybeEntitle(ctx, "baby", "patrick", "user", "userName", "dance")
	assert.Empty(t, ctx.log.infoString())
	assert.Contains(t, ctx.log.errString(), `Could not entitle user "patrick" to app "dance", error: 404 Not Found`)
}

// common method to test getting basic entitlements
func checkGetEntitlementReturns(t *testing.T, entity, rType, rID string) {
	entH := func(t *testing.T, req *tstReq) *tstReply {
		output := `{"items": [{ "Entitlements" : "bar"}]}`
		return &tstReply{output: output, contentType: "application/json"}
	}
	idH := func(t *testing.T, req *tstReq) *tstReply {
		output := fmt.Sprintf(`{"resources": [{ "userName" : "foo", "displayName" : "foo", "id": "%s"}]}`, rID)
		return &tstReply{output: output, contentType: "application/json"}
	}
	entPath := "entitlements/definitions/" + strings.ToLower(rType) + "/" + rID
	paths := map[string]tstHandler{
		"GET" + vidmBasePath + "scim/" + rType: idH,
		"GET" + vidmBasePath + entPath:         entH,
		"POST" + vidmTokenPath:                 tstClientCredGrant}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "entitlement", "get", entity, "foo"); ctx != nil {
		assert.Contains(t, ctx.info, "Entitlements: bar")
	}

}
