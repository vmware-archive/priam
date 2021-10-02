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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli"
	. "github.com/vmware/priam/core"
	"github.com/vmware/priam/mocks"
	. "github.com/vmware/priam/testaid"
	. "github.com/vmware/priam/util"

	"gopkg.in/yaml.v2"
)

const (
	yamlUsersFile           = "../resources/newusers.yaml"
	goodAccessToken         = "travolta.was.here"
	goodAuthHeader          = "Bearer " + goodAccessToken
	badAccessToken          = "travolta.has.gone"
	goodIdToken             = "this.is.me"
	vidmBasePathTenantInUrl = "/SAAS" + vidmBasePath
	healthApi               = "GET" + vidmBasePathTenantInUrl + "health"
)

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

/* Strict match. */
func (ctx *tstCtx) assertOnlyInfoEquals(expected string) {
	assert.Empty(ctx.t, ctx.err, "Error message should be empty")
	assert.Equal(ctx.t, ctx.info, expected, "Info message should equal '"+expected+"'")
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
	return fmt.Sprintf("%s    %s: Bearer\n    %s: %s\n    %s: %s\n", tstSrvTgt(url),
		accessTokenTypeOption, accessTokenOption, goodAccessToken, idTokenOption, goodIdToken)
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

/* Helper function to run the given command with a non-existent target and a valid authorization
   header. Params are the testing pointer and the list of arguments for the command.
   Returns the test output context.
*/
func testCliCommand(t *testing.T, args ...string) *tstCtx {
	return runner(newTstCtx(t, tstSrvTgtWithAuth("http://frozen.site")), args...)
}

/* Helper function to run the given command with a non-existent target and a valid authorization
   header. Checks mocked expectations. Params are the testing pointer, a mocked interface
   and the list of arguments for the command. Returns the test output context.
*/
func testMockCommand(t *testing.T, m *mock.Mock, args ...string) (ctx *tstCtx) {
	ctx = runner(newTstCtx(t, tstSrvTgtWithAuth("http://frozen.site")), args...)
	m.AssertExpectations(t)
	return
}

func runWithServer(t *testing.T, paths map[string]TstHandler, args ...string) *tstCtx {
	srv := StartTstServer(t, paths)
	defer srv.Close()
	return runner(newTstCtx(t, tstSrvTgtWithAuth(srv.URL)), args...)
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
	paths := map[string]TstHandler{healthApi: ErrorHandler(500, "favourite 500 error")}
	ctx := runWithServer(t, paths, "target", "radio2.example.com", "sassoon")
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
	paths := map[string]TstHandler{healthApi: healthHandler(false)}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon")
	ctx.assertOnlyErrContains("Reply from " + srv.URL + " does not meet health check")
}

func TestAddNewTargetSucceedsIfHealthCheckSucceeds(t *testing.T) {
	paths := map[string]TstHandler{healthApi: healthHandler(true)}
	srv := StartTstServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon")
	ctx.assertOnlyInfoContains("new target is: sassoon, " + srv.URL)

	cfg := &Config{}
	err := yaml.Unmarshal([]byte(ctx.cfg), cfg)
	require.NoError(t, err)
	require.Equal(t, nil, cfg.Targets[cfg.CurrentTarget][InsecureSkipVerifyOption])
	require.Equal(t, false, cfg.OptionAsBool(InsecureSkipVerifyOption))
	require.Equal(t, "", cfg.Option(InsecureSkipVerifyOption))
}

func TestAddNewHttpsTargetSucceedsWithInsecureSkipVerify(t *testing.T) {
	paths := map[string]TstHandler{healthApi: healthHandler(true)}
	srv := StartTstTLSServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", "--insecure-skip-verify", srv.URL, "sassoon")
	ctx.assertOnlyInfoContains("new target is: sassoon, " + srv.URL)

	cfg := &Config{}
	err := yaml.Unmarshal([]byte(ctx.cfg), cfg)
	require.NoError(t, err)
	require.Equal(t, true, cfg.Targets[cfg.CurrentTarget][InsecureSkipVerifyOption])
	require.Equal(t, true, cfg.OptionAsBool(InsecureSkipVerifyOption))
}

func TestAddNewHttpsTargetFailsWithoutInsecureSkipVerify(t *testing.T) {
	paths := map[string]TstHandler{healthApi: healthHandler(true)}
	srv := StartTstTLSServer(t, paths)
	defer srv.Close()
	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "target", srv.URL, "sassoon")
	ctx.assertOnlyErrContains("Error checking health of " + srv.URL)
	ctx.assertOnlyErrContains("Error checking health of " + srv.URL + ": Get " + srv.URL + "/SAAS/jersey/manager/api/health: x509: certificate signed by unknown authority")
}

func TestHealth(t *testing.T) {
	paths := map[string]TstHandler{healthApi: healthHandler(true)}
	runWithServer(t, paths, "health").assertOnlyInfoContains("allOk")
}

func TestExitIfHealthFails(t *testing.T) {
	paths := map[string]TstHandler{healthApi: ErrorHandler(404, "test health")}
	runWithServer(t, paths, "health").assertOnlyErrContains("test health")
}

// -- test login -----------------------------------------------------------------------------

func TestCanNotLoginWithNoTarget(t *testing.T) {
	ctx := runner(newTstCtx(t, " "), "login", "c", "s")
	ctx.assertOnlyErrContains("no target set")
}

// Helper to setup mock for the token service
func setupTokenServiceMock() *mocks.TokenGrants {
	tokenServiceFactoryMock := new(mocks.TokenServiceFactory)
	tokenServiceMock := new(mocks.TokenGrants)
	tokenServiceFactoryMock.On("GetTokenService", mock.Anything, mock.Anything, mock.Anything).Return(tokenServiceMock)
	tokenServiceFactory = tokenServiceFactoryMock
	return tokenServiceMock
}

func TestCanHandleBadUserLoginReply(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("LoginSystemUser", mock.Anything, "john", "travolta").Return(TokenInfo{}, errors.New("crap"))
	ctx := testMockCommand(t, &tsMock.Mock, "login", "john", "travolta")
	ctx.assertOnlyErrContains("Error getting access token: crap")
}

func TestCanHandleBadOAuthClientCredentialsGrantReply(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("ClientCredentialsGrant", mock.Anything, "john", "travolta").Return(TokenInfo{}, errors.New("crap"))
	ctx := testMockCommand(t, &tsMock.Mock, "login", "-c", "john", "travolta")
	ctx.assertOnlyErrContains("Error getting access token: crap")
}

// Helper function for OAuth2 login
func assertLoginSucceeded(t *testing.T, tokenType string, ctx *tstCtx) {
	assert.Contains(t, ctx.cfg, accessTokenTypeOption+": "+tokenType)
	assert.Contains(t, ctx.cfg, accessTokenOption+": "+goodAccessToken)
	ctx.assertOnlyInfoContains("Access token saved")
}

func TestCanLoginAsOAuthClient(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("ClientCredentialsGrant", mock.Anything, "john", "travolta").
		Return(TokenInfo{AccessTokenType: "Bearer", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login", "-c", "john", "travolta")
	assertLoginSucceeded(t, "Bearer", ctx)
}

func TestPromptForOauthClient(t *testing.T) {
	consoleInput = strings.NewReader("john")
	getRawPassword = func() ([]byte, error) { return []byte("travolta"), nil }
	tsMock := setupTokenServiceMock()
	tsMock.On("ClientCredentialsGrant", mock.Anything, "john", "travolta").
		Return(TokenInfo{AccessTokenType: "Bearer", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login", "-c")
	assertLoginSucceeded(t, "Bearer", ctx)
	ctx.assertOnlyInfoContains("Client ID: ")
	ctx.assertOnlyInfoContains("Secret: ")
}

func TestCanLoginAsSystemUser(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("LoginSystemUser", mock.Anything, "john", "travolta").
		Return(TokenInfo{AccessTokenType: "HZN", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login", "john", "travolta")
	assertLoginSucceeded(t, "HZN", ctx)
}

func TestCanLoginAsUserPromptPassword(t *testing.T) {
	getRawPassword = func() ([]byte, error) { return []byte("TravoLta"), nil }
	tsMock := setupTokenServiceMock()
	tsMock.On("LoginSystemUser", mock.Anything, "jon", "TravoLta").
		Return(TokenInfo{AccessTokenType: "HZN", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login", "jon")
	assertLoginSucceeded(t, "HZN", ctx)
	ctx.assertOnlyInfoContains("Password: ")
}

func TestPromptForSystemUserCreds(t *testing.T) {
	consoleInput = strings.NewReader("olivia")
	getRawPassword = func() ([]byte, error) { return []byte("grease"), nil }
	tsMock := setupTokenServiceMock()
	tsMock.On("LoginSystemUser", mock.Anything, "olivia", "grease").
		Return(TokenInfo{AccessTokenType: "HZN", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login")
	assertLoginSucceeded(t, "HZN", ctx)
	ctx.assertOnlyInfoContains("Password: ")
	ctx.assertOnlyInfoContains("Username: ")
}

// roll our own panic handler since AssertExpectations also panics and assert.Panics can't distinguish which panic.
func catchPanic(f func()) (msg string) {
	defer func() {
		if err := recover(); err != nil {
			msg = fmt.Sprintf("%v", err)
		}
	}()
	f()
	return
}

type badReader struct{}

func (br badReader) Read(p []byte) (int, error) {
	return 0, errors.New("bad console input")
}

func TestPanicIfCantGetUserName(t *testing.T) {
	consoleInput = badReader{}
	getRawPassword = func() ([]byte, error) { return []byte(""), nil }
	tsMock := setupTokenServiceMock()
	msg := catchPanic(func() { testMockCommand(t, &tsMock.Mock, "login") })
	assert.Equal(t, "bad console input", msg)
	tsMock.AssertExpectations(t)
}

func TestPanicIfCantGetPassword(t *testing.T) {
	consoleInput = strings.NewReader("rizzo")
	getRawPassword = func() ([]byte, error) { return nil, errors.New("getRawPassword failed") }
	tsMock := setupTokenServiceMock()
	msg := catchPanic(func() { testMockCommand(t, &tsMock.Mock, "login") })
	assert.Equal(t, "getRawPassword failed", msg)
	tsMock.AssertExpectations(t)
}

// test authcode login with and without user hint
func TestCanHandleBadAuthCodeGrantReply(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("AuthCodeGrant", mock.Anything, "").Return(TokenInfo{}, errors.New("crap"))
	ctx := testMockCommand(t, &tsMock.Mock, "login", "-a")
	ctx.assertOnlyErrContains("Error getting tokens via browser: crap")
}

func TestAuthCodeGrant(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("AuthCodeGrant", mock.Anything, "").
		Return(TokenInfo{AccessTokenType: "Bearer", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login", "-a")
	assertLoginSucceeded(t, "Bearer", ctx)
}

func TestAuthCodeGrantWithHint(t *testing.T) {
	tsMock := setupTokenServiceMock()
	tsMock.On("AuthCodeGrant", mock.Anything, "elsa").
		Return(TokenInfo{AccessTokenType: "Bearer", AccessToken: goodAccessToken}, nil)
	ctx := testMockCommand(t, &tsMock.Mock, "login", "-a", "elsa")
	assertLoginSucceeded(t, "Bearer", ctx)
}

// -- test logout

func TestLogout(t *testing.T) {
	ctx := testCliCommand(t, "logout")
	assert.NotContains(t, ctx.cfg, accessTokenOption)
	assert.NotContains(t, ctx.cfg, goodAccessToken)
	ctx.assertOnlyInfoContains("Access token removed")
}

// -- common CLI checks

func TestPanicOnUnsupportedOptionType(t *testing.T) {
	assert.Panics(t, func() { makeOptionMap(nil, []cli.Flag{cli.IntSliceFlag{}}, "n", "v") })
}

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

// Helper to setup mock for the user service
func setupUsersServiceMock() *mocks.DirectoryService {
	usersServiceMock := new(mocks.DirectoryService)
	usersService = usersServiceMock
	return usersServiceMock
}

func TestCanAddUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("AddEntity", mock.Anything, &BasicUser{Name: "elsa", Given: "", Family: "", Email: "", Pwd: "frozen"}).Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "add", "elsa", "frozen")
}

func TestCanGetUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("DisplayEntity", mock.Anything, "elsa").Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "get", "elsa")
}

func TestCanDeleteUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("DeleteEntity", mock.Anything, "elsa").Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "delete", "elsa")
}

func TestCanListUsersWithCount(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("ListEntities", mock.Anything, 10, "").Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "list", "--count", "10")
}

func TestCanListUsersWithFilter(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("ListEntities", mock.Anything, 0, "filter").Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "list", "--filter", "filter")
}

func TestCanUpdateUserPassword(t *testing.T) {
	newpassword := "friendsforever"
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("UpdateEntity", mock.Anything, "elsa", &BasicUser{Pwd: newpassword}).Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "password", "elsa", newpassword)
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
	ctx := testMockCommand(t, &usersServiceMock.Mock, "user", "password", "elsa")
	ctx.assertOnlyInfoContains("Passwords didn't match. Try again.")
}

func TestCanUpdateUserInfo(t *testing.T) {
	newemail, newgiven, newfamily := "elsa@arendelle.com", "elsa", "frozen"
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("UpdateEntity", mock.Anything, "elsa", &BasicUser{Name: "elsa", Family: newfamily, Email: newemail, Given: newgiven}).Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "update", "elsa", "--given", newgiven, "--family", newfamily, "--email", newemail)
}

func TestLoadUsersFromYamlFile(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("LoadEntities", mock.Anything, yamlUsersFile).Return()
	testMockCommand(t, &usersServiceMock.Mock, "user", "load", yamlUsersFile)
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
	testMockCommand(t, &groupsServiceMock.Mock, "group", "get", "friendsforever")
}

func TestCanListGroups(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("ListEntities", mock.Anything, 0, "").Return(nil)
	testMockCommand(t, &groupsServiceMock.Mock, "group", "list")
}

func TestCanListGroupsWithCount(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("ListEntities", mock.Anything, 13, "").Return(nil)
	testMockCommand(t, &groupsServiceMock.Mock, "group", "list", "--count", "13")
}

func TestCanListGroupsWithFilter(t *testing.T) {
	groupsServiceMock := setupGroupsServiceMock()
	groupsServiceMock.On("ListEntities", mock.Anything, 0, "myfilter").Return(nil)
	testMockCommand(t, &groupsServiceMock.Mock, "group", "list", "--filter", "myfilter")
}

func TestCanAddMemberToGroup(t *testing.T) {
	groupServiceMock := setupGroupsServiceMock()
	groupServiceMock.On("UpdateMember", mock.Anything, "friendsforever", "sven", false).Return()
	testMockCommand(t, &groupServiceMock.Mock, "group", "member", "friendsforever", "sven")
}

func TestCanRemoveMemberFromGroup(t *testing.T) {
	groupServiceMock := setupGroupsServiceMock()
	groupServiceMock.On("UpdateMember", mock.Anything, "friendsforever", "sven", true).Return()
	testMockCommand(t, &groupServiceMock.Mock, "group", "member", "--delete", "friendsforever", "sven")
}

// - Policies

func TestCanListAccessPolicies(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		assert.Empty(t, req.Input)
		assert.Equal(t, req.Authorization, goodAuthHeader)
		return &TstReply{Output: `{"items": [ {"name": "default_access_policy_set"} ]}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{"GET/SAAS/jersey/manager/api/accessPolicies": h}
	ctx := runWithServer(t, paths, "policies")
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
	ctx := runWithServer(t, paths, "schema", unknownSchema)
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
	ctx := runWithServer(t, paths, "schema", schemaType)
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
	ctx := runWithServer(t, paths, "localuserstore")
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
	ctx := runWithServer(t, paths, "localuserstore", "showLocalUserStore=false")
	ctx.assertOnlyInfoContains("---- Local User Store configuration ----")
	ctx.assertOnlyInfoContains(`{"showLocalUserStore": false}`)
}

func TestErrorWhenCannotSetLocalUserStoreConfiguration(t *testing.T) {
	paths := map[string]TstHandler{
		"PUT/SAAS/jersey/manager/api/localuserstore": ErrorHandler(500, "error test")}
	ctx := runWithServer(t, paths, "localuserstore", "showLocalUserStore=false")
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
	testMockCommand(t, &rolesServiceMock.Mock, "role", "get", "friendsforever")
}

func TestCanDisplayAllRoles(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("ListEntities", mock.Anything, 0, "").Return()
	testMockCommand(t, &rolesServiceMock.Mock, "role", "list")
}

func TestCanDisplayAllRolesWithCountAndFilter(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("ListEntities", mock.Anything, 2, "filter").Return()
	testMockCommand(t, &rolesServiceMock.Mock, "role", "list", "--count", "2", "--filter", "filter")
}

func TestCanAddMemberToRole(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("UpdateMember", mock.Anything, "friendsforever", "sven", false).Return()
	testMockCommand(t, &rolesServiceMock.Mock, "role", "member", "friendsforever", "sven")
}

func TestCanRemoveMemberFromRole(t *testing.T) {
	rolesServiceMock := setupRolesServiceMock()
	rolesServiceMock.On("UpdateMember", mock.Anything, "friendsforever", "sven", true).Return()
	testMockCommand(t, &rolesServiceMock.Mock, "role", "member", "--delete", "friendsforever", "sven")
}

// - Tenant
func TestGetTenantConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/tenants/tenant/tenantName/config": h}
	ctx := runWithServer(t, paths, "tenant", "tenantName")
	ctx.assertOnlyInfoContains("---- Tenant configuration ----")
}

func TestSetTenantConfiguration(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: `{}`, ContentType: "application/json"}
	}
	paths := map[string]TstHandler{
		"GET/SAAS/jersey/manager/api/tenants/tenant/tenantName/config": h}
	ctx := runWithServer(t, paths, "tenant", "tenantName")
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
	testMockCommand(t, &appsServiceMock.Mock, "app", "get", "makesnow")
}

func TestCanDeleteApp(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("Delete", mock.Anything, "makesnow").Return()
	testMockCommand(t, &appsServiceMock.Mock, "app", "delete", "makesnow")
}

func TestCanListApps(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("List", mock.Anything, 0, "").Return()
	testMockCommand(t, &appsServiceMock.Mock, "app", "list")
}

func TestCanListAppsWithCountAndFilter(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("List", mock.Anything, 2, "filter").Return()
	testMockCommand(t, &appsServiceMock.Mock, "app", "list", "--count", "2", "--filter", "filter")
}

func TestCanPublishAnAppWithASpecificManifest(t *testing.T) {
	appsServiceMock := setupAppsServiceMock()
	appsServiceMock.On("Publish", mock.Anything, "my-manifest.yaml").Return()
	testMockCommand(t, &appsServiceMock.Mock, "app", "add", "my-manifest.yaml")
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

// - Oauth2 Application Templates

// Helper to setup mock for the app template service
func setupTemplateServiceMock() *mocks.OauthResource {
	templServiceMock := new(mocks.OauthResource)
	templateService = templServiceMock
	return templServiceMock
}

func TestCanGetTemplate(t *testing.T) {
	templServiceMock := setupTemplateServiceMock()
	templServiceMock.On("Get", mock.Anything, "makesnow").Return()
	testMockCommand(t, &templServiceMock.Mock, "template", "get", "makesnow")
}

// Helper to create template map
func templateInfo(name, scope string, accessTokenTTL int) map[string]interface{} {
	return map[string]interface{}{"scope": scope, "accessTokenTTL": accessTokenTTL,
		"authGrantTypes": "authorization_code", "displayUserGrant": false,
		"resourceUuid": "00000000-0000-0000-0000-000000000000", "tokenType": "Bearer",
		"appProductId": name, "redirectUri": "horizonapi://oauth2",
		"refreshTokenTTL": 2628000, "length": 32}
}

func TestCanAddTemplateWithDefaults(t *testing.T) {
	templServiceMock := setupTemplateServiceMock()
	templServiceMock.On("Add", mock.Anything, "olaf", templateInfo("olaf", "user profile email", 480)).Return()
	testMockCommand(t, &templServiceMock.Mock, "template", "add", "olaf")
}

func TestCanAddTemplateWithOptions(t *testing.T) {
	templServiceMock := setupTemplateServiceMock()
	templServiceMock.On("Add", mock.Anything, "olaf", templateInfo("olaf", "snow", 0)).Return()
	testMockCommand(t, &templServiceMock.Mock, "template", "add", "--scope", "snow", "--accessTokenTTL", "0", "olaf")
}

func TestCanDeleteTemplate(t *testing.T) {
	templServiceMock := setupTemplateServiceMock()
	templServiceMock.On("Delete", mock.Anything, "sven").Return()
	testMockCommand(t, &templServiceMock.Mock, "template", "delete", "sven")
}

func TestCannotDeleteTemplateIfNoNameSpecified(t *testing.T) {
	testCliCommand(t, "template", "delete").assertInfoErrContains("USAGE", "Input Error: at least 1 arguments must be given")
}

func TestCanListTemplates(t *testing.T) {
	templServiceMock := setupTemplateServiceMock()
	templServiceMock.On("List", mock.Anything).Return()
	testMockCommand(t, &templServiceMock.Mock, "template", "list")
}

// - Oauth2 Clients

// Helper to setup mock for the client service
func setupClientServiceMock() *mocks.OauthResource {
	clntServiceMock := new(mocks.OauthResource)
	clientService = clntServiceMock
	return clntServiceMock
}

func TestCanGetClient(t *testing.T) {
	clntServiceMock := setupClientServiceMock()
	clntServiceMock.On("Get", mock.Anything, "makesnow").Return()
	testMockCommand(t, &clntServiceMock.Mock, "client", "get", "makesnow")
}

// Helper to create client map
func clientInfo(name, scope string, accessTokenTTL int) map[string]interface{} {
	return map[string]interface{}{"accessTokenTTL": accessTokenTTL,
		"authGrantTypes": "authorization_code", "clientId": name, "displayUserGrant": false,
		"inheritanceAllowed": false, "internalSystemClient": false,
		"redirectUri": "horizonapi://oauth2", "refreshTokenTTL": 2628000, "rememberAs": "",
		"resourceUuid": "00000000-0000-0000-0000-000000000000", "scope": scope,
		"secret": "", "strData": "", "tokenLength": 32, "tokenType": "Bearer",
	}
}

func TestCanAddClientWithDefaults(t *testing.T) {
	clntServiceMock := setupClientServiceMock()
	clntServiceMock.On("Add", mock.Anything, "olaf", clientInfo("olaf", "user profile email", 480)).Return()
	testMockCommand(t, &clntServiceMock.Mock, "client", "add", "olaf")
}

func TestCanAddClientWithOptions(t *testing.T) {
	clntServiceMock := setupClientServiceMock()
	clntServiceMock.On("Add", mock.Anything, "olaf", clientInfo("olaf", "snow", 0)).Return()
	testMockCommand(t, &clntServiceMock.Mock, "client", "add", "--scope", "snow", "--accessTokenTTL", "0", "olaf")
}

func TestCanDeleteClient(t *testing.T) {
	clntServiceMock := setupClientServiceMock()
	clntServiceMock.On("Delete", mock.Anything, "sven").Return()
	testMockCommand(t, &clntServiceMock.Mock, "client", "delete", "sven")
}

func TestCannotDeleteClientIfNoNameSpecified(t *testing.T) {
	testCliCommand(t, "client", "delete").assertInfoErrContains("USAGE", "Input Error: at least 1 arguments must be given")
}

func TestCanListClients(t *testing.T) {
	clntServiceMock := setupClientServiceMock()
	clntServiceMock.On("List", mock.Anything).Return()
	testMockCommand(t, &clntServiceMock.Mock, "client", "list")
}

func TestCanRegisterCliClient(t *testing.T) {
	expectedCliClientRegistration := map[string]interface{}{"clientId": "github.com-vmware-priam", "secret": "not-a-secret",
		"accessTokenTTL": 60 * 60, "authGrantTypes": "authorization_code refresh_token", "displayUserGrant": false,
		"redirectUri": TokenCatcherURI, "refreshTokenTTL": 60 * 60 * 24 * 30, "scope": "openid user profile email admin"}

	clntServiceMock := setupClientServiceMock()
	clntServiceMock.On("Add", mock.Anything, cliClientID, expectedCliClientRegistration).Return()
	testMockCommand(t, &clntServiceMock.Mock, "client", "register")
}

// Token

func TestCanValidateIDToken(t *testing.T) {
	tokenServiceMock := setupTokenServiceMock()
	tokenServiceMock.On("ValidateIDToken", mock.Anything, goodIdToken).Return(nil)
	testMockCommand(t, &tokenServiceMock.Mock, "token", "validate")
}

func TestCanValidateABadIDToken(t *testing.T) {
	tokenServiceMock := setupTokenServiceMock()
	tokenServiceMock.On("ValidateIDToken", mock.Anything, goodIdToken).Return(errors.New("bad bad bad"))
	testMockCommand(t, &tokenServiceMock.Mock, "token", "validate")
}

func TestPrintErrorOnValidateIfNotLoggedIn(t *testing.T) {
	tokenServiceMock := setupTokenServiceMock()
	runner(newTstCtx(t, tstSrvTgt("http://not.logged.in")), "token", "validate")
	tokenServiceMock.AssertExpectations(t)
}

func TestPassEmptyTokenForValidationIfNoIDTokenSaved(t *testing.T) {
	tokenServiceMock := setupTokenServiceMock()
	tokenServiceMock.On("ValidateIDToken", mock.Anything, "").Return(errors.New("bad bad bad"))
	serverTarget := fmt.Sprintf("%s    accesstokentype: Bearer\n    accesstoken: %s\n", tstSrvTgt("http://no.id.token.site"), goodAccessToken)
	runner(newTstCtx(t, serverTarget), "token", "validate")
	tokenServiceMock.AssertExpectations(t)
}

const expectedAwsStsEndpoint = "https://sts.amazonaws.com"

func TestCanUpdateAWSCredentialsInDefaultCredFile(t *testing.T) {
	cfgFile := filepath.Join(os.Getenv("HOME"), ".aws/credentials")
	tokenServiceMock := setupTokenServiceMock()
	tokenServiceMock.On("UpdateAWSCredentials", mock.Anything, goodIdToken, "space-hound", expectedAwsStsEndpoint, cfgFile, "priam").Return(nil)
	testMockCommand(t, &tokenServiceMock.Mock, "token", "aws", "space-hound")
}

func TestCanUpdateAWSCredentialsInExplicitCredFile(t *testing.T) {
	cfgFile := "/var/tmp/my-cred-file"
	tokenServiceMock := setupTokenServiceMock()
	tokenServiceMock.On("UpdateAWSCredentials", mock.Anything, goodIdToken, "space-messenger", expectedAwsStsEndpoint, cfgFile, "priam").Return(nil)
	testMockCommand(t, &tokenServiceMock.Mock, "token", "aws", "-c", cfgFile, "space-messenger")
}

func TestCanUpdateAWSCredentialsInExplicitCredFileAndProfile(t *testing.T) {
	cfgFile := "/var/tmp/my-cred-file"
	tokenServiceMock := setupTokenServiceMock()
	tokenServiceMock.On("UpdateAWSCredentials", mock.Anything, goodIdToken, "space-hound", expectedAwsStsEndpoint, cfgFile, "kazak").Return(nil)
	testMockCommand(t, &tokenServiceMock.Mock, "token", "aws", "-c", cfgFile, "-p", "kazak", "space-hound")
}
