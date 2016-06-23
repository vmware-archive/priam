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
	"strings"
	"testing"
)

const YAML_USERS_FILE = "../resources/newusers.yaml"

type UsersServiceMock struct {
	mock.Mock
}

type tstCtx struct {
	appName, cfg, input, info, err string
	printResults                   bool
}

func (ctx *tstCtx) printOut() *tstCtx {
	ctx.printResults = true
	return ctx
}

func newTstCtx(cfg string) *tstCtx {
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
	return &tstCtx{appName: "testapp", cfg: StringOrDefault(cfg, sampleCfg)}
}

func tstSrvTgt(url string) string {
	return fmt.Sprintf("---\ncurrenttarget: 1\ntargets:\n  1:\n    host: %s\n", url)
}

// Helpers to get health handler
func healthHandler(status bool) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		assert.Empty(t, req.Input)
		if status {
			return &TstReply{Output: `{"allOk":true}`, ContentType: "application/json"}
		}
		return &TstReply{Output: `{"somethingelse":true}`, ContentType: "application/json"}
	}
}

// in these tests the clientID is "john" and the client secret is "travolta"
// Adapted from tests written by Fanny, who apparently likes John Travolta
func tstClientCredGrant(t *testing.T, req *TstReq) *TstReply {
	const tokenReply = `{"token_type": "Bearer", "access_token": "testvalidtoken"}`
	const basicAuthJohnTravolta = "Basic am9objp0cmF2b2x0YQ=="
	assert.Equal(t, basicAuthJohnTravolta, req.Authorization)
	assert.Equal(t, "grant_type=client_credentials", req.Input)
	return &TstReply{Output: tokenReply}
}

func tstSrvTgtWithAuth(url string) string {
	return tstSrvTgt(url) + "    clientid: john\n    clientsecret: travolta\n"
}

func runner(t *testing.T, ctx *tstCtx, args ...string) *tstCtx {
	cfgFile := WriteTempFile(t, ctx.cfg)
	defer CleanupTempFile(cfgFile)
	args = append([]string{ctx.appName}, args...)
	infoW, errW := bytes.Buffer{}, bytes.Buffer{}
	Priam(args, cfgFile.Name(), strings.NewReader(ctx.cfg), &infoW, &errW)
	_, err := cfgFile.Seek(0, 0)
	assert.Nil(t, err)
	contents, err := ioutil.ReadAll(cfgFile)
	assert.Nil(t, err)
	ctx.cfg, ctx.info, ctx.err = string(contents), infoW.String(), errW.String()
	if ctx.printResults {
		fmt.Printf("----------------config:\n%s\n", ctx.cfg)
		fmt.Printf("----------------info:\n%s\n", ctx.info)
		fmt.Printf("----------------error:\n%s\n", ctx.err)
	}
	require.NotNil(t, ctx)
	return ctx
}

// help usage
func TestHelp(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "help"); ctx != nil {
		assert.Contains(t, ctx.info, "USAGE")
	}
}

// unknown flag should not crash the app
func TestUnknownFlagOption(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "--unknowflag", "2", "user", "list"); ctx != nil {
		assert.Contains(t, ctx.info, "USAGE")
	}
}

// help user load usage includes password and does not require target
func TestHelpUserLoad(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "user", "help", "load"); ctx != nil {
		assert.Contains(t, ctx.info, "user load")
	}
}

func TestHelpUserLoadOption(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "user", "load", "-h"); ctx != nil {
		assert.Contains(t, ctx.info, "user load")
	}
}

func TestAppAnyName(t *testing.T) {
	ctx, name := newTstCtx(""), "welcome_back_kotter"
	ctx.appName = name
	if ctx := runner(t, ctx, "-h"); ctx != nil {
		assert.Contains(t, ctx.info, name)
	}
}

// should not pick a target if none is set
func TestTargetNoCurrent(t *testing.T) {
	var targetYaml string = `---
targets:
  radio:
    host: https://radio.example.com
`
	if ctx := runner(t, newTstCtx(targetYaml), "target"); ctx != nil {
		assert.Equal(t, "no target set\n", ctx.info)
	}
}

// should use the current target if one is set
func TestTargetCurrent(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "target"); ctx != nil {
		assert.Contains(t, ctx.info, "radio1.example.com")
	}
}

// should fail gracefully if no config exists
func TestTargetNoConfig(t *testing.T) {
	if ctx := runner(t, newTstCtx(" "), "target"); ctx != nil {
		assert.Contains(t, ctx.info, "no target set")
	}
}

// should not require access to server if target forced
func TestTargetForced(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "target", "-f", "https://bad.example.com"); ctx != nil {
		assert.Contains(t, ctx.info, "bad.example.com")
	}
}

// should add https to target url if needed
func TestTargetAddHttps(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "target", "-f", "bad.example.com"); ctx != nil {
		assert.Contains(t, ctx.info, "https://bad.example.com")
	}
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
	if ctx := runner(t, newTstCtx(""), "targets"); ctx != nil {
		assert.Equal(t, expectedSorted, ctx.info)
	}
}

func TestReuseExistingTargetHostWithoutName(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "target", "radio2.example.com"); ctx != nil {
		assert.Contains(t, ctx.info, "new target is: staging, https://radio2.example.com")
	}
}

func TestReuseExistingTargetByName(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "target", "staging"); ctx != nil {
		assert.Contains(t, ctx.info, "new target is: staging, https://radio2.example.com")
	}
}

func TestAddNewTargetWithName(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "target", "-f", "radio2.example.com", "sassoon"); ctx != nil {
		assert.Contains(t, ctx.info, "new target is: sassoon, https://radio2.example.com")
	}
}

func TestAddNewTargetFailsIfHealthCheckFails(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": ErrorHandler(500, "favourite 500 error")}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "target", "radio2.example.com", "sassoon"); ctx != nil {
		assert.Contains(t, ctx.err, "Error checking health of https://radio2.example.com")
	}
}

func TestAddNewTargetFailsIfHealthCheckDoesNotContainAllOkTrue(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": healthHandler(false)}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon"); ctx != nil {
		assert.Contains(t, ctx.err, "Reply from "+srv.URL+" does not meet health check")
	}
}

func TestAddNewTargetSucceedsIfHealthCheckSucceeds(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": healthHandler(true)}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon"); ctx != nil {
		assert.Contains(t, ctx.info, "new target is: sassoon, "+srv.URL)
	}
}

func TestHealth(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": healthHandler(true)}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "health"); ctx != nil {
		assert.Contains(t, ctx.info, "allOk")
	}
}

func TestExitIfHealthFails(t *testing.T) {
	paths := map[string]TstHandler{"GET" + vidmBasePath + "health": ErrorHandler(404, "test health")}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "health"); ctx != nil {
		assert.Contains(t, ctx.err, "test health")
	}
}

// -- Login
func TestCanNotLoginWithNoTarget(t *testing.T) {
	if ctx := runner(t, newTstCtx(" "), "login", "c", "s"); ctx != nil {
		assert.Contains(t, ctx.err, "no target set")
	}
}

func TestCanNotLoginWithTargetSetButNoOauthCreds(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "login"); ctx != nil {
		assert.Contains(t, ctx.err, "at least 1 arguments must be given")
	}
}

func TestCanHandleBadLoginReply(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		assert.NotEmpty(t, req.Input)
		return &TstReply{Output: "crap"}
	}
	paths := map[string]TstHandler{"POST" + vidmTokenPath: h}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "login", "john", "travolta"); ctx != nil {
		assert.Contains(t, ctx.err, "invalid")
	}
}

func TestCanLogin(t *testing.T) {
	paths := map[string]TstHandler{"POST" + vidmTokenPath: tstClientCredGrant}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "login", "john", "travolta"); ctx != nil {
		assert.Contains(t, ctx.cfg, "clientid: john")
		assert.Contains(t, ctx.cfg, "clientsecret: travolta")
		assert.Contains(t, ctx.info, "clientID and clientSecret saved")
	}
}

// -- common CLI checks

func TestCanNotRunACommandWithTooManyArguments(t *testing.T) {
	if ctx := runner(t, newTstCtx(""), "app", "get", "too", "many", "args"); ctx != nil {
		assert.Contains(t, ctx.err, "at most 1 arguments can be given")
	}
}

// -- user commands
func TestCanNotIssueUserCommandWithTooManyArguments(t *testing.T) {
	for _, command := range []string{"add", "update", "list", "get", "delete", "load", "password"} {
		if ctx := runner(t, newTstCtx(""), "user", command, "too", "many", "args"); ctx != nil {
			assert.Contains(t, ctx.err, "Input Error: at most")
		}
	}
}

// Helper function to start the test HTTP server and run the given command
// @param args the list of arguments for the command
// @return The mock for users service.
func testCliCommand(t *testing.T, args ...string) *tstCtx {
	paths := map[string]TstHandler{"POST" + vidmTokenPath: tstClientCredGrant}
	srv := StartTstServer(t, paths)
	return runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), args...)
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
	if ctx := testCliCommand(t, "user", "add", "elsa", "frozen"); ctx != nil {
		assert.Contains(t, ctx.info, "User 'elsa' successfully added")
	}
	usersServiceMock.AssertExpectations(t)
}

func TestDisplayErrorWhenAddUserFails(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("AddEntity",
		mock.Anything, &BasicUser{Name: "elsa", Given: "", Family: "", Email: "", Pwd: "frozen"}).Return(errors.New("test"))
	if ctx := testCliCommand(t, "user", "add", "elsa", "frozen"); ctx != nil {
		assert.Contains(t, ctx.err, "Error creating user 'elsa': test")
	}
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
	usersServiceMock.On("LoadEntities", mock.Anything, YAML_USERS_FILE).Return()
	testCliCommand(t, "user", "load", YAML_USERS_FILE)
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
		return &TstReply{Output: `{"items": [ {"name": "default_access_policy_set"} ]}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"POST" + vidmTokenPath:                       tstClientCredGrant,
		"GET/SAAS/jersey/manager/api/accessPolicies": h}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "policies")
	assert.NotNil(t, ctx)
	assert.Contains(t, ctx.info, "---- Access Policies ----\nitems:\n- name: default_access_policy_set")
}

// - Schema
func TestCannotGetSchemaIfNoTypeSpecified(t *testing.T) {
	ctx := testCliCommand(t, "schema")
	assert.NotNil(t, ctx)
	assert.Contains(t, ctx.err, "Input Error: at least 1 arguments must be given")
}

func TestCannotGetSchemaforUnknownType(t *testing.T) {
	unknownSchema := "olaf"
	paths := map[string]TstHandler{
		"POST" + vidmTokenPath:                                                                tstClientCredGrant,
		"GET/SAAS/jersey/manager/api/scim/Schemas?filter=name+eq+%22" + unknownSchema + "%22": ErrorHandler(404, "test schema")}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "schema", unknownSchema)
	assert.NotNil(t, ctx)
	assert.Contains(t, ctx.err, "test schema")
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
		"POST" + vidmTokenPath:                                                             tstClientCredGrant,
		"GET/SAAS/jersey/manager/api/scim/Schemas?filter=name+eq+%22" + schemaType + "%22": h}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "schema", schemaType)
	assert.Contains(t, ctx.info, "---- Schema for "+schemaType+" ----\nattributes:")
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
		"POST" + vidmTokenPath:                       tstClientCredGrant,
		"GET/SAAS/jersey/manager/api/localuserstore": h}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "localuserstore")
	assert.Contains(t, ctx.info, "---- Local User Store configuration ----")
	assert.Contains(t, ctx.info, "name: Test Local Users")
	assert.Contains(t, ctx.info, "showLocalUserStore: true")
	assert.Contains(t, ctx.info, "uuid: \"123\"")
}

func TestCanSetLocalUserStoreConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{"showLocalUserStore": false}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"POST" + vidmTokenPath:                       tstClientCredGrant,
		"PUT/SAAS/jersey/manager/api/localuserstore": h}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "localuserstore", "showLocalUserStore=false")
	assert.Contains(t, ctx.info, "---- Local User Store configuration ----")
	assert.Contains(t, ctx.info, `{"showLocalUserStore": false}`)
}

func TestErrorWhenCannotSetLocalUserStoreConfiguration(t *testing.T) {
	paths := map[string]TstHandler{
		"POST" + vidmTokenPath:                       tstClientCredGrant,
		"PUT/SAAS/jersey/manager/api/localuserstore": ErrorHandler(500, "error test")}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "localuserstore", "showLocalUserStore=false")
	assert.Contains(t, ctx.err, "error test")
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
		"POST" + vidmTokenPath:                                         tstClientCredGrant,
		"GET/SAAS/jersey/manager/api/tenants/tenant/tenantName/config": h}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "tenant", "tenantName")
	assert.Contains(t, ctx.info, "---- Tenant configuration ----")
}

func TestSetTenantConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"POST" + vidmTokenPath:                                         tstClientCredGrant,
		"GET/SAAS/jersey/manager/api/tenants/tenant/tenantName/config": h}
	srv := StartTstServer(t, paths)
	ctx := runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), "tenant", "tenantName")
	assert.Contains(t, ctx.info, "---- Tenant configuration ----")
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
