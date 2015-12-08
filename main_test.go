package main

import (
	//"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

var sampleCfg string = `---
currenttarget: 1
targets:
  radio:
    host: https://radio.workspace.com
  1:
    host: https://radio.workspaceair.com
  staging:
    host: https://radio.hwslabs.com
`

type context struct {
	cfgin,        cfgout,        stdin,        stdout,        stderr string
	expectError                                                      bool
}

func newContext(cfg string) *context {
	return &context{cfgin: stringOrDefault(cfg, sampleCfg)}
}

// Build a context with expected error
func newErrorContext(cfg string) *context {
	ctx := newContext(cfg)
	ctx.expectError = true
	return ctx
}

func closeDelete(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}

type reqInfo struct {
	reply,        mtype,        expect string
}

func StartTestServer(paths map[string]reqInfo) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {

		println("processing request: ", r.URL.Path)
		if rbody, err := ioutil.ReadAll(r.Body); err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %v", err), 404)
		} else if info, ok := paths[r.URL.Path]; !ok {
			http.Error(w, fmt.Sprintf("bad path: %s", r.URL.Path), 404)
		} else if info.expect == "" || info.expect == string(rbody) {

			println("   - replying with", info.reply)

			w.Header().Set("Content-Type", stringOrDefault(info.mtype, "application/json"))

			io.WriteString(w, info.reply)
		}
	}
	return httptest.NewServer(http.HandlerFunc(handler))
}

func tstSrvTgt(url string) string {
	return fmt.Sprintf("---\ntargets:\n  1:\n    host: %s\n", url)
}

func runner(t *testing.T, ctx *context, args ...string) *context {
	var err error
	f := map[string]*os.File{"c": nil, "i": nil, "o": nil, "e": nil}
	for k, _ := range f {
		if f[k], err = ioutil.TempFile("", "wks-test-" + k + "-"); !assert.Nil(t, err) {
			return nil
		}
		defer closeDelete(f[k])
	}
	for k, v := range map[string]*string{"c": &ctx.cfgin, "i": &ctx.stdin} {
		if _, err = f[k].Write([]byte(*v)); !assert.Nil(t, err) {
			return nil
		}
	}
	orgIn, orgErr, orgOut := os.Stdin, os.Stderr, os.Stdout
	defer func() {
		os.Stdin, os.Stderr, os.Stdout = orgIn, orgErr, orgOut
	}()
	os.Stdin, os.Stderr, os.Stdout = f["i"], f["e"], f["o"]
	os.Args = append([]string{"wks", "--config", f["c"].Name()}, args...)
	main()
	for k, v := range map[string]*string{"c": &ctx.cfgout, "o": &ctx.stdout, "e": &ctx.stderr} {
		if _, err = f[k].Seek(0, 0); !assert.Nil(t, err) {
			return nil
		} else if contents, err := ioutil.ReadAll(f[k]); !assert.Nil(t, err) {
			return nil
		} else {
			*v = string(contents)
		}
	}
	if ctx.expectError {
		if !assert.NotEmpty(t, ctx.stderr) {
			return nil
		}
	} else {
		if !assert.Empty(t, ctx.stderr) {
			return nil
		}
	}
	return ctx
}

// help usage
func TestHelp(t *testing.T) {
	if ctx := runner(t, newContext(""), "help"); ctx != nil {
		assert.Contains(t, ctx.stdout, "USAGE")
	}
}

// should pick a target if none is set
func TestTargetNoCurrent(t *testing.T) {
	var targetYaml string = `---
targets:
  radio:
    host: https://radio.workspaceair.com
`
	if ctx := runner(t, newContext(targetYaml), "target"); ctx != nil {
		assert.Contains(t, ctx.cfgout, "radio.workspaceair.com")
	}
}

// should use the current target if one is set
func TestTargetCurrent(t *testing.T) {
	if ctx := runner(t, newContext(""), "target"); ctx != nil {
		assert.Contains(t, ctx.stdout, "radio.workspaceair.com")
	}
}

// should fail gracefully if the config file does not exist
func TestTargetNoConfig(t *testing.T) {
	if ctx := runner(t, newContext(" "), "target"); ctx != nil {
		assert.Contains(t, ctx.stdout, "no target set")
	}
}

// should fail gracefully if no appConfig exists
func TestTargetNoConfig(t *testing.T) {
	if ctx := runner(t, newContext(" "), "target"); ctx != nil {
		assert.Contains(t, ctx.stdout, "no target set")
	}
}

// should not require access to server if target forced
func TestTargetForced(t *testing.T) {
	if ctx := runner(t, newContext(""), "target", "-f", "https://bad.example.com"); ctx != nil {
		assert.Contains(t, ctx.stdout, "bad.example.com")
	}
}

// should add https to target url if needed
func TestTargetAddHttps(t *testing.T) {
	if ctx := runner(t, newContext(""), "target", "-f", "bad.example.com"); ctx != nil {
		assert.Contains(t, ctx.stdout, "https://bad.example.com")
	}
}

func TestTargets(t *testing.T) {
	if ctx := runner(t, newContext(""), "-d", "targets"); ctx != nil {
		assert.Contains(t, ctx.stdout, "staging")
		assert.Contains(t, ctx.stdout, "radio")
		assert.Contains(t, ctx.stdout, "https://radio.workspaceair.com")
	}
}

func TestHealth(t *testing.T) {
	paths := map[string]reqInfo{"/SAAS/jersey/manager/api/health": reqInfo{reply: "allOk"}}
	srv := StartTestServer(paths)
	if ctx := runner(t, newContext(tstSrvTgt(srv.URL)), "health"); ctx != nil {
		assert.Contains(t, ctx.stdout, "allOk")
	}
}

// -- Entitlements methods

func TestGetEntitlementWithNoArgsShowsHelp(t *testing.T) {
	if ctx := runner(t, newContext(""), "entitlement"); ctx != nil {
		assert.Contains(t, ctx.stdout, "USAGE")
	}
}

func TestGetEntitlementWithNoTypeShowsError(t *testing.T) {
	if ctx := runner(t, newErrorContext(""), "entitlement", "get"); ctx != nil {
		assert.Contains(t, ctx.stderr, "at least 2 arguments must be specified")
	}
}

func TestGetEntitlementWithNoNameShowsError(t *testing.T) {
	types := [...]string{"user", "app", "group"}
	for i := range types {
		if ctx := runner(t, newErrorContext(""), "entitlement", "get", types[i]); ctx != nil {
			assert.Contains(t, ctx.stderr, "at least 2 arguments must be specified")
		}
	}
}

// common method to test entitlement
func checkGetEntitlementReturnsError(t *testing.T, entity string) {
	if ctx := runner(t, newErrorContext(""), "entitlement", "get", entity, "foo"); ctx != nil {
		assert.Contains(t, ctx.stderr, "error: invalid_client")
	}

}

// Don't know how to mock functions (and avoid global)
// So just check we will get an error
func TestGetEntitlementForUser(t *testing.T) {
	checkGetEntitlementReturnsError(t, "user")
}

func TestGetEntitlementForGroup(t *testing.T) {
	checkGetEntitlementReturnsError(t, "group")
}

func TestGetEntitlementForApp(t *testing.T) {
	checkGetEntitlementReturnsError(t, "app")
}


// -- Login
func TestCanNotLoginWithNoTarget(t *testing.T) {
	if ctx := runner(t, newErrorContext(" "), "login"); ctx != nil {
		assert.Contains(t, ctx.stderr, "Error: no target set")
	}
}

func TestCanNotLoginWithTargetSetButNoOauthCreds(t *testing.T) {
	if ctx := runner(t, newErrorContext(sampleCfg), "login"); ctx != nil {
		assert.Contains(t, ctx.stderr, "must supply clientID and clientSecret on the command line")
	}
}

func TestCanHandleBadLoginReply(t *testing.T) {
	paths := map[string]reqInfo{"/SAAS/API/1.0/oauth2/token": reqInfo{reply: "crap"}}
	srv := StartTestServer(paths)
	if ctx := runner(t, newErrorContext(tstSrvTgt(srv.URL)), "login", "john", "travolta"); ctx != nil {
		assert.Contains(t, ctx.stderr, "invalid")
	}
}

func TestCanLogin(t *testing.T) {
	paths := map[string]reqInfo{"/SAAS/API/1.0/oauth2/token": reqInfo{reply: `{"access_token" : "ABC", "token_type" : "TestTokenType"}`}}
	srv := StartTestServer(paths)
	if ctx := runner(t, newContext(tstSrvTgt(srv.URL)), "login", "john", "travolta"); ctx != nil {
		assert.Contains(t, ctx.stdout, "clientID and clientSecret saved")
	}
}


// -- common CLI checks

func TestCanNotRunACommandWithTooManyArguments(t *testing.T) {
	if ctx := runner(t, newErrorContext(sampleCfg), "app", "get", "too", "many", "args"); ctx != nil {
		assert.Contains(t, ctx.stderr, "at most 1 arguments can be specified")
	}
}
