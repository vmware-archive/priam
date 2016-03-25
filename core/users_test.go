package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
	"fmt"
)

const (
	DEFAULT_USERNAME = "john"
	DEFAULT_GROUP_NAME = "saturday-night-fever"

	DEFAULT_GET_USER_URL = "GET/scim/Users?count=10000&filter=userName+eq+%22" + DEFAULT_USERNAME + "%22"
	DEFAULT_POST_USER_URL = "POST/scim/Users/12345"

	DEFAULT_GET_GROUP_URL = "GET/scim/Groups?count=10000&filter=displayName+eq+%22" + DEFAULT_GROUP_NAME + "%22"
	YAML_USERS_FILE = "../resources/newusers.yaml"
)

var aBasicUser = func() *basicUser { return &basicUser{Name: "john", Given: "travolta"} }

// Helpers to get useful handlers used in the tests
func scimDefaultUserHandler() func(t *testing.T, req *tstReq) *tstReply {
	return func(t *testing.T, req *tstReq) *tstReply {
		output := `{"Resources": [{ "userName" : "` + DEFAULT_USERNAME + `", "id": "12345"}]}`
		return &tstReply{output: output, contentType: "application/json"}
	}
}

func scimDefaultGroupHandler() func(t *testing.T, req *tstReq) *tstReply {
	return func(t *testing.T, req *tstReq) *tstReply {
		output := `{"Resources": [{ "displayName" : "` + DEFAULT_GROUP_NAME + `", "id": "6789"}]}`
		return &tstReply{output: output, contentType: "application/json"}
	}
}

// Tests
func TestSetPassword(t *testing.T) {
	pwdH := func(t *testing.T, req *tstReq) *tstReply {
		assert.Contains(t, req.input, "Password")
		return &tstReply{status: 204}
	}
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users/12345": pwdH,
		DEFAULT_GET_USER_URL:    scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).UpdateEntity(ctx, "john", &basicUser{Pwd: "travolta"})
	assertOnlyInfoContains(t, ctx, `User "john" updated`)
}

func TestSetPasswordFailsIfScimPatchFails(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users/12345": ErrorHandler(404, "error set password"),
		DEFAULT_GET_USER_URL:    scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).UpdateEntity(ctx, "john", &basicUser{Pwd: "travolta"})
	assertErrorContains(t, ctx, "Error updating user \"john\": 404 Not Found\nerror set password")
}

func TestScimGetByNameWhenUserDoesNotExistReturnsError(t *testing.T) {
	errMessage := "test: john does not exist"
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, errMessage)})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	_, err := scimGetByName(ctx, "Users", "userName", "john")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), errMessage)
	}
}

func TestScimGetByNameWhenMultipleUsersReturnsError(t *testing.T) {
	multipleUsersHandler := func(t *testing.T, req *tstReq) *tstReply {
		output := `{"resources": [{ "userName" : "john", "id": "12345"}, { "userName" : "john", "id": "54321"}]}`
		return &tstReply{output: output, contentType: "application/json"}
	}

	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: multipleUsersHandler})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	_, err := scimGetByName(ctx, "Users", "userName", "john")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), "multiple Users found named \"john\"")
	}
}

func TestScimGetByNameWhenNoMatchReturnsError(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"GET/scim/Users?count=10000&filter=userName+eq+%22patrick%22": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	_, err := scimGetByName(ctx, "Users", "userName", "patrick")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), `no Users found named "patrick"`)
	}
}

func TestScimGetIdWithNoIdReturnsError(t *testing.T) {
	noIdHandler := func(t *testing.T, req *tstReq) *tstReply {
		output := `{"resources": [{ "userName" : "john" }]}`
		return &tstReply{output: output, contentType: "application/json"}
	}
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: noIdHandler})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	_, err := scimGetID(ctx, "Users", "userName", "john")
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), `no id returned for "john"`)
	}
}

func TestScimListWithCountAndFilter(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"GET/scim/Users?count=3&filter=myfilter": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).ListEntities(ctx, 3, "myfilter")
	assertOnlyInfoContains(t, ctx, `id: 12345`)
}

func TestScimListFilteredByLabel(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"GET/scim/Users?filter=myfilter": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimList(ctx, 0, "myfilter", "Users", "userName")
	assertOnlyInfoContains(t, ctx, "userName: john")
	assert.NotContains(t, ctx.log.infoString(), "id")
}

func TestScimListWithNonExistingSummaryLabelsPrintsEmpty(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"GET/scim/Users": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimList(ctx, 0, "", "Users", "IDontExist")
	assertOnlyInfoContains(t, ctx, "")
}

func TestScimListReturnsErrorOnInvalidRequest(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"GET/scim/Users?filter=myfilter": ErrorHandler(404, "error scim list")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).ListEntities(ctx, 0, "myfilter")
	assertErrorContains(t, ctx, "Error getting SCIM resources of type Users: 404 Not Found\nerror scim list\n")
}

func TestScimGetNameExists(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).DisplayEntity(ctx, "john")
	assertOnlyInfoContains(t, ctx, "userName: john")
}

func TestScimGetWhenNameDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "error scim get")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).DisplayEntity(ctx, "john")
	assertErrorContains(t, ctx, "Error getting SCIM resource named john of type Users: 404 Not Found\nerror scim get\n")
}

func TestScimAddUser(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).AddEntity(ctx, aBasicUser())
	assertOnlyInfoContains(t, ctx, "username: john")
	assertOnlyInfoContains(t, ctx, "---- add user:  ----")
}

func TestScimAddUserReturnsErrorOnScimError(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users": ErrorHandler(404, "error scim add")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	err := new(SCIMUsersService).AddEntity(ctx, aBasicUser())
	if assert.Error(t, err, "Should have returned an error") {
		assert.Contains(t, err.Error(), "404 Not Found\nerror scim add\n")
	}
}

func TestScimUpdateUserFailedIfUserDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "not found")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).UpdateEntity(ctx, "john", &basicUser{Name: "john", Given: "wayne" })
	assertErrorContains(t, ctx, "Error getting SCIM Users ID of john: 404 Not Found")
}

func TestScimUpdateUserFailedIfPatchCommandFails(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler(),
		// response does not matter, only body could be tested
		"POST/scim/Users/12345": ErrorHandler(404, "error scim patch")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).UpdateEntity(ctx, "john", &basicUser{Name: "john", Given: "johnny" })
	assertErrorContains(t, ctx, `Error updating user "john": 404 Not Found`)
}

// Helper for testing SCIM update
func scimUpdateDefaultUserWith(t *testing.T, name string, updatedUser *basicUser) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler(),
		// response does not matter, only body could be tested
		"POST/scim/Users/12345": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).UpdateEntity(ctx, name, updatedUser)
	assertOnlyInfoContains(t, ctx, fmt.Sprintf("User \"%s\" updated", name))
}

func TestScimUpdateUserName(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &basicUser{Name: "newjohn", Given: "johnny" })
}

func TestScimUpdateUserGivenName(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &basicUser{Name: "john", Given: "johnny" })
}

func TestScimUpdateUserFamilyName(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &basicUser{Name: "john", Family: "wayne" })
}

func TestScimUpdateUserEmail(t *testing.T) {
	scimUpdateDefaultUserWith(t, "john", &basicUser{Name: "john", Email: "j@travolta.com" })
}

func TestScimDeleteFailsIfUserDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL:      ErrorHandler(404, "test error")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimDelete(ctx, "Users", "userName", "john")
	assertErrorContains(t, ctx, "Error getting SCIM Users ID of john: 404 Not Found")
}

func TestScimDeleteFailsIfDeleteCommandFails(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL:      scimDefaultUserHandler(),
		"DELETE/scim/Users/12345": ErrorHandler(404, "error scim delete")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimDelete(ctx, "Users", "userName", "john")
	assertErrorContains(t, ctx, "Error deleting Users john: 404 Not Found\nerror scim delete")
}

func TestScimDelete(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: scimDefaultUserHandler(),
		// response does not matter as long as this is 20x
		"DELETE/scim/Users/12345": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).DeleteEntity(ctx, "john")
	assertOnlyInfoContains(t, ctx, `Users "john" deleted`)
}

func TestScimMemberReturnsWhenNoResourceId(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL: ErrorHandler(404, "error scim members")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimMember(ctx, "Users", "userName", "john", "john", false)
	assertErrorContains(t, ctx, "Error getting SCIM Users ID of john")
}

func TestAddScimMember(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL:  scimDefaultUserHandler(),
		DEFAULT_POST_USER_URL: scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimMember(ctx, "Users", "userName", "john", "john", false)
	assertOnlyInfoContains(t, ctx, "Updated SCIM resource john of type Users\n")
}

func TestRemoveScimMember(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL:  scimDefaultUserHandler(),
		DEFAULT_POST_USER_URL: scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimMember(ctx, "Users", "userName", "john", "john", true)
	assertOnlyInfoContains(t, ctx, "Updated SCIM resource john of type Users\n")
}

func TestRemoveScimMemberReturnsErrorIfScimPatchFailed(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_USER_URL:  scimDefaultUserHandler(),
		DEFAULT_POST_USER_URL: ErrorHandler(404, "error scim patch members")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	scimMember(ctx, "Users", "userName", "john", "john", true)
	assertErrorContains(t, ctx, "Error updating SCIM resource john of type Users")
}

func TestLoadUsersFromYaml(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users": scimDefaultUserHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).LoadEntities(ctx, YAML_USERS_FILE)
	assertOnlyInfoContains(t, ctx, "added user joe1")
}

func TestLoadUsersFromYamlFailedIfAddUserFailed(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users": ErrorHandler(404, "error scim add user")})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).LoadEntities(ctx, YAML_USERS_FILE)
	assertErrorContains(t, ctx, "Error adding user, line 2, name joe1: 404 Not Found")
}

func TestLoadUsersFromYamlFailedIfYamlFileDoesNotExist(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMUsersService).LoadEntities(ctx, "newusers-does-not-exist.yaml")
	assertErrorContains(t, ctx, "could not read file of bulk users")
}

// Tests for GROUPS
// @todo To be put in groups_test.go?

func TestGetGroup(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		DEFAULT_GET_GROUP_URL: scimDefaultGroupHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMGroupsService).DisplayEntity(ctx, DEFAULT_GROUP_NAME)
	assertOnlyInfoContains(t, ctx, "displayName: " + DEFAULT_GROUP_NAME)
}

func TestListGroups(t *testing.T) {
	srv := StartTstServer(t, map[string]tstHandler{
		"GET/scim/Groups?count=3&filter=myfilter": scimDefaultGroupHandler()})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	new(SCIMGroupsService).ListEntities(ctx, 3, "myfilter")
	assertOnlyInfoContains(t, ctx, `id: 6789`)
	assertOnlyInfoContains(t, ctx, "displayName: " + DEFAULT_GROUP_NAME)
}
