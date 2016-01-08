package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSchemaWithNoMediaType(t *testing.T) {
	h := func(t *testing.T, req *tstReq) *tstReply {
		assert.Empty(t, req.accept)
		output := `{"attributes": [{ "name" : "id"}, {"name": "userName"}]}`
		return &tstReply{output: output}
	}
	srv := StartTstServer(t, map[string]tstHandler{"GET/scim/Schemas": h})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	cmdSchema(ctx, "User")
	assert.Empty(t, ctx.log.errString())
	assert.Contains(t, ctx.log.infoString(), "Schema for User")
	assert.Contains(t, ctx.log.infoString(), "attributes")
	assert.Contains(t, ctx.log.infoString(), "name: userName")
}
