package main

import (
	//"bytes"
	//"fmt"
	"github.com/stretchr/testify/assert"
	//"io"
	//"net/http"
	"os"
	"testing"
	//"net/http/httptest"
	"io/ioutil"
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
}

func newContext(cfg string) *context {
	return &context{cfgin: stringOrDefault(cfg, sampleCfg)}
}

func closeDelete(f *os.File) {
	f.Close()
	//os.Remove(f.Name())
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
	return ctx
}

func TestHelp(t *testing.T) {
	if ctx := runner(t, newContext(""), "help"); ctx != nil {
		assert.Empty(t, ctx.stderr)
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
		assert.Empty(t, ctx.stderr)
		assert.Contains(t, ctx.cfgout, "radio.workspaceair.com")
	}
}

// should use the current target if one is set
func TestTargetCurrent(t *testing.T) {
	if ctx := runner(t, newContext(""), "target"); ctx != nil {
		assert.Empty(t, ctx.stderr)
		assert.Contains(t, ctx.stdout, "radio.workspaceair.com")
	}
}

// should fail gracefully if no appConfig exists
func TestTargetNoConfig(t *testing.T) {
	if ctx := runner(t, newContext(" "), "target"); ctx != nil {
		assert.Empty(t, ctx.stderr)
		assert.Contains(t, ctx.stdout, "no target set")
	}
}

/*
// should not require access to server if target forced
func TestTargetForced(t *testing.T) {
	ctx := beforeEach()
	assert.Nil(t, wks([]string{"wks", "target", "-f", "https://bad.example.com"}))
	assert.Contains(t, ctx.outb.String(), "bad.example.com")
}

// should add https to target url if needed
func TestTargetAddHttps(t *testing.T) {
	ctx := beforeEach()
	assert.Nil(t, wks([]string{"wks", "target", "-f", "bad.example.com"}))
	assert.Contains(t, ctx.outb.String(), "https://bad.example.com")
}

// TestTargetNonWorkspace
// TestTargetWithName
// TestTargetWithoutName
// TestHealth

type reqInfo struct {
	path, reply string
}

func StartTestServer(info *reqInfo) (srv *httptest.Server) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != info.path {
			http.Error(w, "bad path", 404)
		} else {
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, info.reply)
		}
	}
	srv = httptest.NewServer(http.HandlerFunc(handler))
	inCfg = []byte(fmt.Sprintf("---\ntargets:\n  1:\n    host: %s\n", srv.URL))
	return
}

func TestHealth(t *testing.T) {
	ctx := beforeEach()
	srv := StartTestServer(&reqInfo{"/SAAS/jersey/manager/api/health", "allOk"})
	defer srv.Close()
	assert.Nil(t, wks([]string{"wks", "health"}))
	assert.Contains(t, ctx.outb.String(), "allOk")
}

func TestTargets(t *testing.T) {
	ctx := beforeEach()
	assert.Nil(t, wks([]string{"wks", "-d", "targets"}))
	assert.Contains(t, ctx.outb.String(), "staging")
	assert.Contains(t, ctx.outb.String(), "radio")
	assert.Contains(t, ctx.outb.String(), "https://radio.workspaceair.com")
}
*/
