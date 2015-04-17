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

func closeDelete(f *os.File) {
	f.Close()
	os.Remove(f.Name())
}

func runner(t *testing.T, config string, input string, args ...string) (cfgout, stdout, errout string) {
	orgIn, orgErr, orgOut := os.Stdin, os.Stderr, os.Stdout
	fc, err := ioutil.TempFile("", "wks-test-c-")
	if !assert.Nil(t, err) {
		return
	}
	defer closeDelete(fc)
	fi, err := ioutil.TempFile("", "wks-test-i-")
	if !assert.Nil(t, err) {
		return
	}
	defer closeDelete(fi)
	fo, err := ioutil.TempFile("", "wks-test-o-")
	if !assert.Nil(t, err) {
		return
	}
	defer closeDelete(fo)
	fe, err := ioutil.TempFile("", "wks-test-e-")
	if !assert.Nil(t, err) {
		return
	}
	defer closeDelete(fe)
	_, err = fc.Write([]byte(config))
	if !assert.Nil(t, err) {
		return
	}
	_, err = fi.Write([]byte(input))
	if !assert.Nil(t, err) {
		return
	}
	os.Stdin, os.Stderr, os.Stdout = fi, fe, fo
	os.Args = append([]string{"wks", "--config", fc.Name()}, args...)
	//append(os.Args, args...)
	main()
	fc.Seek(0, 0)
	fe.Seek(0, 0)
	fo.Seek(0, 0)
	contents, err := ioutil.ReadAll(fc)
	if !assert.Nil(t, err) {
		return
	}
	cfgout = string(contents)
	contents, err = ioutil.ReadAll(fo)
	if !assert.Nil(t, err) {
		return
	}
	stdout = string(contents)
	contents, err = ioutil.ReadAll(fe)
	if !assert.Nil(t, err) {
		return
	}
	errout = string(contents)
	os.Stdin, os.Stderr, os.Stdout = orgIn, orgErr, orgOut
	return
}

func TestHelp(t *testing.T) {
	_, stdo, erro := runner(t, sampleCfg, "", "help")
	assert.Empty(t, erro)
	assert.Contains(t, stdo, "USAGE")
}

/*
// should pick a target if none is set
func TestTargetNoCurrent(t *testing.T) {
	var targetYaml string = `---
targets:
  radio:
    host: https://radio.workspaceair.com
`
	ctx := beforeEach()
	inCfg = []byte(targetYaml)
	assert.Nil(t, wks([]string{"wks", "target"}))
	assert.Contains(t, ctx.outb.String(), "radio.workspaceair.com")
}

// should use the current target if one is set
func TestTargetCurrent(t *testing.T) {
	ctx := beforeEach()
	assert.Nil(t, wks([]string{"wks", "target"}))
	assert.Contains(t, ctx.outb.String(), "radio.workspaceair.com")
	//println(len(ctx.outb), ctx.outb)
	assert.Empty(t, ctx.errb.Bytes())
}

// should fail gracefully if no appConfig exists
func TestTargetNoConfig(t *testing.T) {
	ctx := beforeEach()
	inCfg = []byte{}
	assert.Nil(t, wks([]string{"wks", "target"}))
	assert.Contains(t, ctx.outb.String(), "no target set")
}

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
