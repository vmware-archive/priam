package main

import (
	//"fmt"
	"testing"
	"bytes"
	"github.com/stretchr/testify/assert"
)

type context struct {
	outb, inb, errb bytes.Buffer
}

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

func beforeEach() (ctx *context) {
	ctx = new(context)
	outW = &ctx.outb
	inR = &ctx.inb
	errW = &ctx.errb
	inCfg = []byte{}
	outCfg = []byte{}
	manifest = []byte{}
	inCfg = []byte(sampleCfg)
	return
}

func TestHelp(t *testing.T) {
	ctx := beforeEach()
	assert.Nil(t, wks([]string{"wks", "help"}))
	assert.Contains(t, ctx.outb.String(), "USAGE")
}

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
	assert.Nil(t, wks([]string{"wks", "target", "bad.example.com"}))
	assert.Contains(t, ctx.outb.String(), "https://bad.example.com")
}

// TestTargetNonWorkspace
// TestTargetForce
// TestTargetWithName
// TestTargetWithoutName
// TestTargetAddHttps
// TestHealth




