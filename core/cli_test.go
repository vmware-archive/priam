package core

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"io/ioutil"
	"strings"
	"testing"
)

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
	return &tstCtx{appName: "testapp", cfg: stringOrDefault(cfg, sampleCfg)}
}

func tstSrvTgt(url string) string {
	return fmt.Sprintf("---\ntargets:\n  1:\n    host: %s\n", url)
}

// in these tests the clientID is "john" and the client secret is "travolta"
// Adapted from tests written by Fanny, who apparently likes John Travolta
func tstClientCredGrant(t *testing.T, req *tstReq) *tstReply {
	const tokenReply = `{"token_type": "Bearer", "access_token": "testvalidtoken"}`
	const basicAuthJohnTravolta = "Basic am9objp0cmF2b2x0YQ=="
	assert.Equal(t, basicAuthJohnTravolta, req.authorization)
	assert.Equal(t, "grant_type=client_credentials", req.input)
	return &tstReply{output: tokenReply}
}

func tstSrvTgtWithAuth(url string) string {
	return tstSrvTgt(url) + "    clientid: john\n    clientsecret: travolta\n"
}

func runner(t *testing.T, ctx *tstCtx, args ...string) *tstCtx {
	cfgFile := WriteTempFile(t, ctx.cfg)
	defer CleanupTempFile(cfgFile)
	args = append([]string{ctx.appName, "--config", cfgFile.Name()}, args...)
	infoW, errW := bytes.Buffer{}, bytes.Buffer{}
	Priam(args, strings.NewReader(ctx.cfg), &infoW, &errW)
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

// should pick a target if none is set
func TestTargetNoCurrent(t *testing.T) {
	var targetYaml string = `---
targets:
  radio:
    host: https://radio.example.com
`
	if ctx := runner(t, newTstCtx(targetYaml), "target"); ctx != nil {
		assert.Contains(t, ctx.cfg, "radio.example.com")
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
	if ctx := runner(t, newTstCtx(""), "targets"); ctx != nil {
		assert.Contains(t, ctx.info, "staging")
		assert.Contains(t, ctx.info, "radio")
		assert.Contains(t, ctx.info, "https://radio.example.com")
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

func TestHealth(t *testing.T) {
	h := func(t *testing.T, req *tstReq) *tstReply {
		assert.Empty(t, req.input)
		return &tstReply{output: `{"allOk":true}`, contentType: "application/json"}
	}
	paths := map[string]tstHandler{"GET" + vidmBasePath + "health": h}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "health"); ctx != nil {
		assert.Contains(t, ctx.info, "allOk")
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
	h := func(t *testing.T, req *tstReq) *tstReply {
		assert.NotEmpty(t, req.input)
		return &tstReply{output: "crap"}
	}
	paths := map[string]tstHandler{"POST" + vidmTokenPath: h}
	srv := StartTstServer(t, paths)
	if ctx := runner(t, newTstCtx(tstSrvTgt(srv.URL)), "login", "john", "travolta"); ctx != nil {
		assert.Contains(t, ctx.err, "invalid")
	}
}

func TestCanLogin(t *testing.T) {
	paths := map[string]tstHandler{"POST" + vidmTokenPath: tstClientCredGrant}
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
	paths := map[string]tstHandler{"POST" + vidmTokenPath: tstClientCredGrant}
	srv := StartTstServer(t, paths)
	return runner(t, newTstCtx(tstSrvTgtWithAuth(srv.URL)), args...)
}

// Helper to setup mock for users service
func setupUsersServiceMock() *MockDirectoryService {
	usersServiceMock := new(MockDirectoryService)
	usersService = usersServiceMock
	return usersServiceMock
}

func TestCanAddUser(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("AddEntity", mock.Anything, &basicUser{Name: "elsa", Given: "", Family: "", Email: "", Pwd: "frozen"}).Return(nil)
	if ctx := testCliCommand(t, "user", "add", "elsa", "frozen"); ctx != nil {
		assert.Contains(t, ctx.info, "User 'elsa' successfully added")
	}
	usersServiceMock.AssertExpectations(t)
}

func TestDisplayErrorWhenAddUserFails(t *testing.T) {
	usersServiceMock := setupUsersServiceMock()
	usersServiceMock.On("AddEntity",
		mock.Anything, &basicUser{Name: "elsa", Given: "", Family: "", Email: "", Pwd: "frozen"}).Return(errors.New("test"))
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
