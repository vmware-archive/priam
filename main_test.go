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
	cfgin, cfgout, stdin, stdout, stderr string
	expectError                          bool
}

func newContext(cfg string) *context {
	return &context{cfgin: stringOrDefault(cfg, sampleCfg)}
}

func closeDelete(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}

type reqInfo struct {
	reply, mtype, expect string
}

func StartTestServer(paths map[string]reqInfo) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if rbody, err := ioutil.ReadAll(r.Body); err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %v", err), 404)
		} else if info, ok := paths[r.URL.Path]; !ok {
			http.Error(w, "bad path", 404)
		} else if info.expect == "" || info.expect == string(rbody) {
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
		if f[k], err = ioutil.TempFile("", "wks-test-"+k+"-"); !assert.Nil(t, err) {
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
