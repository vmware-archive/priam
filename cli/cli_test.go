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

package cli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	. "priam/core"
	"priam/mocks"
	. "priam/testaid"
	. "priam/util"
	"testing"
)

const (
	yamlUsersFile   = "../resources/newusers.yaml"
	goodAccessToken = "travolta.was.here"
	goodAuthHeader  = "Bearer " + goodAccessToken
	badAccessToken  = "travolta.has.gone"
)

type UsersServiceMock struct {
	mock.Mock
}

type tstCtx struct {
	t                              *testing.T
	appName, cfg, input, info, err string
	printResults                   bool
}

func (ctx *tstCtx) printOut() *tstCtx {
	ctx.printResults = true
	return ctx
}

func (ctx *tstCtx) assertOnlyInfoContains(expected string) {
	assert.Empty(ctx.t, ctx.err, "Error message should be empty")
	assert.Contains(ctx.t, ctx.info, expected, "Info message should contain '"+expected+"'")
}

func (ctx *tstCtx) assertOnlyErrContains(expected string) {
	assert.Empty(ctx.t, ctx.info, "Info message should be empty")
	assert.Contains(ctx.t, ctx.err, expected, "Error should contain '"+expected+"'")
}

func (ctx *tstCtx) assertInfoErrContains(expectedInfo, expectedErr string) {
	assert.Contains(ctx.t, ctx.info, expectedInfo, "Info message should contain '"+expectedInfo+"'")
	assert.Contains(ctx.t, ctx.err, expectedErr, "Error should contain '"+expectedErr+"'")
}

func newTstCtx(t *testing.T, cfg string) *tstCtx {
	sampleCfg := `---
currenttarget: 1
targets:
  radio:
    host: https://radio.example.com
  1:
    host: https://radio1.example.com
  staging:
    host: https://radio2.example.com
`
	return &tstCtx{t: t, appName: "testapp", cfg: StringOrDefault(cfg, sampleCfg)}
}

func tstSrvTgt(url string) string {
	return fmt.Sprintf("---\ncurrenttarget: 1\ntargets:\n  1:\n    host: %s\n", url)
}

func tstSrvTgtWithAuth(url string) string {
	return fmt.Sprintf("%s    authheader: Bearer %s\n", tstSrvTgt(url), goodAccessToken)
}

func runner(ctx *tstCtx, args ...string) *tstCtx {
	cfgFile := WriteTempFile(ctx.t, ctx.cfg)
	defer CleanupTempFile(cfgFile)
	args = append([]string{ctx.appName}, args...)
	infoW, errW := bytes.Buffer{}, bytes.Buffer{}
	Priam(args, cfgFile.Name(), &infoW, &errW)
	_, err := cfgFile.Seek(0, 0)
	require.Nil(ctx.t, err)
	contents, err := ioutil.ReadAll(cfgFile)
	require.Nil(ctx.t, err)
	ctx.cfg, ctx.info, ctx.err = string(contents), infoW.String(), errW.String()
	if ctx.printResults {
		fmt.Printf("----------------config:\n%s\n", ctx.cfg)
		fmt.Printf("----------------info:\n%s\n", ctx.info)
		fmt.Printf("----------------error:\n%s\n", ctx.err)
	}
	return ctx
}

// -- test help usage -----------------------------------------------------------
func TestHelp(t *testing.T) {
	runner(newTstCtx(t, ""), "help").assertOnlyInfoContains("USAGE")
}

// unknown flag should not crash the app
func TestUnknownFlagOption(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "--unknowflag", "2", "user", "list")
	ctx.assertInfoErrContains("USAGE", "flag provided but not defined: -unknowflag")
}

// help user load usage includes password and does not require target
func TestHelpUserLoad(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "user", "help", "load")
	ctx.assertOnlyInfoContains("user load")
}

func TestHelpUserLoadOption(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "user", "load", "-h")
	ctx.assertOnlyInfoContains("user load")
}

func TestAppAnyName(t *testing.T) {
	ctx, name := newTstCtx(t, ""), "welcome_back_kotter"
	ctx.appName = name
	runner(ctx, "-h").assertOnlyInfoContains(name)
}

// -- test target command -----------------------------------------------------

// should not pick a target if none is set
func TestTargetNoCurrent(t *testing.T) {
	var targetYaml string = `---
targets:
  radio:
    host: https://radio.example.com
`
	ctx := runner(newTstCtx(t, targetYaml), "target")
	ctx.assertOnlyInfoContains("no target set\n")
}

// should use the current target if one is set
func TestTargetCurrent(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "target")
	ctx.assertOnlyInfoContains("radio1.example.com")
}

// should fail gracefully if no config exists
func TestTargetNoConfig(t *testing.T) {
	ctx := runner(newTstCtx(t, " "), "target")
	ctx.assertOnlyInfoContains("no target set")
}

// should not require access to server if target forced
func TestTargetForced(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "target", "-f", "https://bad.example.com")
	ctx.assertOnlyInfoContains("bad.example.com")
}

// should add https to target url if needed
func TestTargetAddHttps(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "target", "-f", "bad.example.com")
	ctx.assertOnlyInfoContains("https://bad.example.com")
}

func TestTargets(t *testing.T) {
	expectedSorted := `name: 1
host: https://radio1.example.com

name: radio
host: https://radio.example.com

name: staging
host: https://radio2.example.com

current target is: 1, https://radio1.example.com
`
	ctx := runner(newTstCtx(t, ""), "targets")
	ctx.assertOnlyInfoContains(expectedSorted)

}

func TestReuseExistingTargetHostWithoutName(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "target", "radio2.example.com")
	ctx.assertOnlyInfoContains("new target is: staging, https://radio2.example.com")
}

func TestReuseExistingTargetByName(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "target", "staging")
	ctx.assertOnlyInfoContains("new target is: staging, https://radio2.example.com")
}

func TestAddNewTargetWithName(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "target", "-f", "radio2.example.com", "sassoon")
	ctx.assertOnlyInfoContains("new target is: sassoon, https://radio2.example.com")
}

func TestAddNewTargetFailsIfHealthCheckFails(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": ErrorHandler(500, "favourite 500 error")}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", "radio2.example.com", "sassoon")
	ctx.assertOnlyErrContains("Error checking health of https://radio2.example.com")
}

// Helper health handler
func healthHandler(status bool) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		assert.Empty(t, req.Input)
		if status {
			return &TstReply{Output: `{"allOk":true}`, ContentType: "application/json"}
		}
		return &TstReply{Output: `{"somethingelse":true}`, ContentType: "application/json"}
	}
}

func TestAddNewTargetFailsIfHealthCheckDoesNotContainAllOk(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": healthHandler(false)}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon")
	ctx.assertOnlyErrContains("Reply from " + srv.URL + " does not meet health check")
}

func TestAddNewTargetSucceedsIfHealthCheckSucceeds(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": healthHandler(true)}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon")
	ctx.assertOnlyInfoContains("new target is: sassoon, " + srv.URL)
}

func TestHealth(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": healthHandler(true)}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "health")
	ctx.assertOnlyInfoContains("allOk")
}

func TestExitIfHealthFails(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": ErrorHandler(404, "test health")}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "health")
	ctx.assertOnlyErrContains("test health")
}

// -- test login -----------------------------------------------------------------------------

func TestCanNotLoginWithNoTarget(t *testing.T) {
	ctx := runner(newTstCtx(t, " "), "login", "c", "s")
	ctx.assertOnlyErrContains("no target set")
}

func TestCanNotLoginWithTargetSetButNoOauthCreds(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "login", "-c")
	ctx.assertInfoErrContains("USAGE", "at least 1 arguments must be given")
}

func TestCanNotLoginWithTargetSetButUserCreds(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "login")
	ctx.assertInfoErrContains("USAGE", "at least 1 arguments must be given")
}

func badLoginReply(t *testing.T, req *TstReq) *TstReply {
	assert.NotEmpty(t, req.Input)
	return &TstReply{Output: "crap"}
}

func TestCanHandleBadUserLoginReply(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{"POST" + vidmLoginPath: badLoginReply})
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john", "travolta")
	ctx.assertOnlyErrContains("invalid")
}

func TestCanHandleBadOAuthLoginReply(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{"POST" + vidmTokenPath: badLoginReply})
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "-c", "john", "travolta")
	ctx.assertOnlyErrContains("invalid")
}

// in these tests the clientID is "john" and the client secret is "travolta"
// Adapted from tests written by Fanny, who apparently likes John Travolta
func tstClientCredGrant(t *testing.T, req *TstReq) *TstReply {
	assert.Equal(t, "Basic am9objp0cmF2b2x0YQ==", req.Authorization)
	assert.Equal(t, "grant_type=client_credentials", req.Input)
	return &TstReply{Output: `{"token_type": "Bearer", "access_token": "` + goodAccessToken + `"}`}
}

func TestCanLoginAsOAuthClient(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{"POST" + vidmTokenPath: tstClientCredGrant})
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "-c", "john", "travolta")
	assert.Contains(t, ctx.cfg, "authheader: Bearer "+goodAccessToken)
	ctx.assertOnlyInfoContains("Access token saved")
}

func tstUserLogin(t *testing.T, req *TstReq) *TstReply {
	assert.Contains(t, req.Input, `"username": "john"`)
	assert.Contains(t, req.Input, `"password": "travolta"`)
	assert.Contains(t, req.Input, `"issueToken": true`)
	return &TstReply{Output: `{"admin": false, "sessionToken": "` + goodAccessToken + `"}`}
}

func TestCanLoginAsUser(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{"POST" + vidmLoginPath: tstUserLogin})
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john", "travolta")
	assert.Contains(t, ctx.cfg, "authheader: HZN "+goodAccessToken)
	ctx.assertOnlyInfoContains("Access token saved")
}

func TestCanLoginAsUserPromptPassword(t *testing.T) {
	srv := StartTstServer(t, map[string]TstHandler{"POST" + vidmLoginPath: tstUserLogin})
	defer srv.Close()
	getRawPassword = func() ([]byte, error) { return []byte("travolta"), nil }
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john")
	assert.Contains(t, ctx.cfg, "authheader: HZN "+goodAccessToken)
	ctx.assertOnlyInfoContains("Access token saved")
}

// -- test logout

func TestLogout(t *testing.T) {
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth("http://grease.com")), "logout")
	assert.NotContains(t, ctx.cfg, "authheader")
	assert.NotContains(t, ctx.cfg, goodAccessToken)
	ctx.assertOnlyInfoContains("Access token removed")
}

// -- common CLI checks

func TestCanNotRunACommandWithTooManyArguments(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "app", "get", "too", "many", "args")
	ctx.assertInfoErrContains("USAGE", "at most 1 arguments can be given")
}

// -- user commands
func TestCanNotIssueUserCommandWithTooManyArguments(t *testing.T) {
	for _, command := range []string{"add", "update", "list", "get", "delete", "load", "password"} {
		ctx := runner(newTstCtx(t, ""), "user", command, "too", "many", "args")
		ctx.assertInfoErrContains("USAGE", "Input Error: at most")
	}
}

// Helper function to run the given command
// @param args the list of arguments for the command
// @return The test output context.
func testCliCommand(t *testing.T, args ...string) *tstCtx {
	return runner(newTstCtx(t, tstSrvTgtWithAuth("http://frozen.site")), args...)
}

// Helper to setup mock for the user service
func setupUsersServiceMock() *mocks.DirectoryService {
	usersServiceMock := new(mocks.DirectoryService)
	usersService = usersServiceMock
	return usersServiceMock
}

func TestCanAddUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("AddEntity", mock.Anything, &BasicUser{Name: "elsa", Given: "", Family: "", Email: "", Pwd: "frozen"}).Return(nil)
	ctx := testCliCommand(t, "user", "add", "elsa", "frozen")
	ctx.assertOnlyInfoContains("User 'elsa' successfully added")
	usersServiceMock.AssertExpectations(t)
}

func TestDisplayErrorWhenAddUserFails(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("AddEntity",
		mock.Anything, &BasicUser{Name: "elsa", Given: "", Family: "", Email: "", Pwd: "frozen"}).Return(errors.New("test"))
	ctx := testCliCommand(t, "user", "add", "elsa", "frozen")
	assert.Contains(t, ctx.err, "Error creating user 'elsa': test")
	usersServiceMock.AssertExpectations(t)
}

func TestCanGetUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("DisplayEntity", mock.Anything, "elsa").Return()
	testCliCommand(t, "user", "get", "elsa")
	usersServiceMock.AssertExpectations(t)
}

func TestCanDeleteUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("DeleteEntity", mock.Anything, "elsa").Return()
	testCliCommand(t, "user", "delete", "elsa")
	usersServiceMock.AssertExpectations(t)
}

func TestCanListUsersWithCount(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("ListEntities", mock.Anything, 10, "").Return()
	testCliCommand(t, "user", "list", "--count", "10")
	usersServiceMock.AssertExpectations(t)
}

func TestCanListUsersWithFilter(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("ListEntities", mock.Anything, 0, "filter").Return()
	testCliCommand(t, "user", "list", "--filter", "filter")
	usersServiceMock.AssertExpectations(t)
}

func TestCanUpdateUserPassword(t *testing.T) {
	newpassword := "friendsforever"
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("UpdateEntity", mock.Anything, "elsa", &BasicUser{Pwd: newpassword}).Return()
	testCliCommand(t, "user", "password", "elsa", newpassword)
	usersServiceMock.AssertExpectations(t)
}

func TestCanUpdateUserPasswordPromptedWithTypo(t *testing.T) {
	newpassword, pwdCount := "friendsforever", 0
	getRawPassword = func() ([]byte, error) {
		if pwdCount = pwdCount + 1; pwdCount == 2 {
			return []byte("hans-not-friend"), nil
		}
		return []byte(newpassword), nil
	}
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("UpdateEntity", mock.Anything, "elsa", &BasicUser{Pwd: newpassword}).Return()
	ctx := testCliCommand(t, "user", "password", "elsa")
	ctx.assertOnlyInfoContains("Passwords didn't match. Try again.")
	usersServiceMock.AssertExpectations(t)
}

func TestCanUpdateUserInfo(t *testing.T) {
	newemail := "elsa@arendelle.com"
	newgiven := "elsa"
	newfamily := "frozen"
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("UpdateEntity", mock.Anything, "elsa", &BasicUser{Name: "elsa", Family: newfamily, Email: newemail, Given: newgiven}).Return()
	testCliCommand(t, "user", "update", "elsa", "--given", newgiven, "--family", newfamily, "--email", newemail)
	usersServiceMock.AssertExpectations(t)
}

func TestLoadUsersFromYamlFile(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("LoadEntities", mock.Anything, yamlUsersFile).Return()
	testCliCommand(t, "user", "load", yamlUsersFile)
	usersServiceMock.AssertExpectations(t)
}

// - Groups

// Helper to setup mock for the user service
func setupGroupsServiceMock() *mocks.DirectoryService {
	groupsServiceMock := new(mocks.DirectoryService)
	groupsService = groupsServiceMock
	return groupsServiceMock
}

func TestCanGetGroup(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("DisplayEntity", mock.Anything, "friendsforever").Return(nil)
	testCliCommand(t, "group", "get", "friendsforever")
	groupsServiceMock.AssertExpectations(t)
}

func TestCanListGroups(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("ListEntities", mock.Anything, 0, "").Return(nil)
	testCliCommand(t, "group", "list")
	groupsServiceMock.AssertExpectations(t)
}

func TestCanListGroupsWithCount(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("ListEntities", mock.Anything, 13, "").Return(nil)
	testCliCommand(t, "group", "list", "--count", "13")
	groupsServiceMock.AssertExpectations(t)
}

func TestCanListGroupsWithFilter(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("ListEntities", mock.Anything, 0, "myfilter").Return(nil)
	testCliCommand(t, "group", "list", "--filter", "myfilter")
	groupsServiceMock.AssertExpectations(t)
}

// - Policies

func TestCanListAccessPolicies(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		assert.Empty(t, req.Input)
		assert.Equal(t, req.Authorization, goodAuthHeader)
		return &TstReply{Output: `{"items": [ {"name": "default_access_policy_set"} ]}`, ContentType: "application/json"}
	}
	srv := StartTstServer(t, map[string]TstHandler{"GET/SAAS/jersey/manager/api/accessPolicies": h})
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "policies")
	ctx.assertOnlyInfoContains("---- Access Policies ----\nitems:\n- name: default_access_policy_set")
}

func TestCantListAccessPoliciesWithoutAuth(t *testing.T) {
	runner(newTstCtx(t, ""), "policies").assertOnlyErrContains("No access token")
}

// - Schema
func TestCannotGetSchemaIfNoTypeSpecified(t *testing.T) {
	ctx := testCliCommand(t, "schema")
	ctx.assertInfoErrContains("USAGE", "Input Error: at least 1 arguments must be given")
}

func TestCannotGetSchemaforUnknownType(t *testing.T) {
	unknownSchema := "olaf"
	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/scim/Schemas?filter=name+eq+%22" + unknownSchema + "%22": ErrorHandler(404, "test schema")}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "schema", unknownSchema)
	ctx.assertOnlyErrContains("test schema")
}

func TestCanGetSchema(t *testing.T) {
	for _, schemaType := range []string{"User", "Group", "Role", "PasswordState", "ServiceProviderConfig"} {
		t.Logf("Get schema for '%s'", schemaType)
		canGetSchemaFor(t, schemaType)
	}
}

func canGetSchemaFor(t *testing.T, schemaType string) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{ "attributes": [], "name": "test", "schema": "urn:scim:schemas:core:1.0"}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/scim/Schemas?filter=name+eq+%22" + schemaType + "%22": h}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "schema", schemaType)
	ctx.assertOnlyInfoContains("---- Schema for " + schemaType + " ----\nattributes:")
}

// - User store
func TestCanGetLocalUserStoreConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{

	"name": "Test Local Users",
	"showLocalUserStore": true,
	"syncClient": null,
	"userDomainInfo": {},
    "userStoreNameUsedForAuth": false,
	"uuid": "123"
		}`,
			ContentType: "application/json"}
	}

	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/localuserstore": h}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "localuserstore")
	ctx.assertOnlyInfoContains("---- Local User Store configuration ----")
	ctx.assertOnlyInfoContains("name: Test Local Users")
	ctx.assertOnlyInfoContains("showLocalUserStore: true")
	ctx.assertOnlyInfoContains("uuid: \"123\"")
}

func TestCanSetLocalUserStoreConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{"showLocalUserStore": false}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"PUT/SAAS/jersey/manager/api/localuserstore": h}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "localuserstore", "showLocalUserStore=false")
	ctx.assertOnlyInfoContains("---- Local User Store configuration ----")
	ctx.assertOnlyInfoContains(`{"showLocalUserStore": false}`)
}

func TestErrorWhenCannotSetLocalUserStoreConfiguration(t *testing.T) {
	paths := map[string]TstHandler{
		"PUT/SAAS/jersey/manager/api/localuserstore": ErrorHandler(500, "error test")}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "localuserstore", "showLocalUserStore=false")
	ctx.assertOnlyErrContains("error test")
}

// - Roles

// Helper to setup mock for the roles service
func setupRolesServiceMock() *mocks.DirectoryService {
	rolesServiceMock := new(mocks.DirectoryService)
	rolesService = rolesServiceMock
	return rolesServiceMock
}

func TestCanGetRole(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("DisplayEntity", mock.Anything, "friendsforever").Return()
	testCliCommand(t, "role", "get", "friendsforever")
	rolesServiceMock.AssertExpectations(t)
}

func TestCanDisplayAllRoles(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("ListEntities", mock.Anything, 0, "").Return()
	testCliCommand(t, "role", "list")
	rolesServiceMock.AssertExpectations(t)
}

func TestCanDisplayAllRolesWithCountAndFilter(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("ListEntities", mock.Anything, 2, "filter").Return()
	testCliCommand(t, "role", "list", "--count", "2", "--filter", "filter")
	rolesServiceMock.AssertExpectations(t)
}

// - Tenant
func TestGetTenantConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/tenants/tenant/tenantName/config": h}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "tenant", "tenantName")
	ctx.assertOnlyInfoContains("---- Tenant configuration ----")
}

func TestSetTenantConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/tenants/tenant/tenantName/config": h}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), "tenant", "tenantName")
	ctx.assertOnlyInfoContains("---- Tenant configuration ----")
}

// - Apps

// Helper to setup mock for the apps service
func setupAppsServiceMock() *mocks.ApplicationService {
	appsServiceMock := new(mocks.ApplicationService)
	appsService = appsServiceMock
	return appsServiceMock
}

func TestCanGetApp(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("Display", mock.Anything, "makesnow").Return()
	testCliCommand(t, "app", "get", "makesnow")
	appsServiceMock.AssertExpectations(t)
}

func TestCanDeleteApp(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("Delete", mock.Anything, "makesnow").Return()
	testCliCommand(t, "app", "delete", "makesnow")
	appsServiceMock.AssertExpectations(t)
}

func TestCanListApps(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("List", mock.Anything, 0, "").Return()
	testCliCommand(t, "app", "list")
	appsServiceMock.AssertExpectations(t)
}

func TestCanListAppsWithCountAndFilter(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("List", mock.Anything, 2, "filter").Return()
	testCliCommand(t, "app", "list", "--count", "2", "--filter", "filter")
	appsServiceMock.AssertExpectations(t)
}

func TestCanPublishAnApp(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("Publish", mock.Anything, "").Return()
	testCliCommand(t, "app", "add")
	appsServiceMock.AssertExpectations(t)
}

func TestCanPublishAnAppWithASpecificManifest(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("Publish", mock.Anything, "my-manifest.yaml").Return()
	testCliCommand(t, "app", "add", "my-manifest.yaml")
	appsServiceMock.AssertExpectations(t)
}

// - Entitlements

func TestGetEntitlementWithNoArgsShowsHelp(t *testing.T) {
	ctx := runner(newTstCtx(t, ""), "entitlement")
	ctx.assertOnlyInfoContains("USAGE")
}

func TestGetEntitlementWithNoTypeShowsError(t *testing.T) {
	ctx := runner(newTstCtx(t, " "), "entitlement", "get")
	ctx.assertInfoErrContains("USAGE", "at least 2 arguments must be given")
}

func TestGetEntitlementWithNoNameShowsError(t *testing.T) {
	types := []string{"user", "app", "group"}
	for i := range types {
		ctx := runner(newTstCtx(t, " "), "entitlement", "get", types[i])
		ctx.assertInfoErrContains("USAGE", "at least 2 arguments must be given")
	}
}

func TestGetEntitlementWithWrongTypeShowsError(t *testing.T) {
	ctx := runner(newTstCtx(t, " "), "entitlement", "get", "actor", "swayze")
	ctx.assertInfoErrContains("USAGE", "First parameter of 'get' must be user, group or app")
}
