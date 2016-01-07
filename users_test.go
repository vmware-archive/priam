package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetPassword(t *testing.T) {
	pwdH := func(t *testing.T, req *tstReq) *tstReply {
		assert.Contains(t, req.input, "Password")
		return &tstReply{status: 204}
	}
	idH := func(t *testing.T, req *tstReq) *tstReply {
		output := `{"resources": [{ "userName" : "john", "id": "12345"}]}`
		return &tstReply{output: output, contentType: "application/json"}
	}
	srv := StartTstServer(t, map[string]tstHandler{
		"POST/scim/Users/12345": pwdH,
		"GET/scim/Users":        idH})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	cmdSetPassword(ctx, "john", "travolta")
	assert.Empty(t, ctx.log.errString())
	assert.Contains(t, ctx.log.infoString(), `User "john" updated`)
}
