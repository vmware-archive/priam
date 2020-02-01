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

package core

import (
	"github.com/stretchr/testify/assert"
	. "github.com/vmware/priam/testaid"
	. "github.com/vmware/priam/util"
	"net/http/httptest"
	"testing"
)

func TestGetSchemaWithNoMediaType(t *testing.T) {
	h := func(t *testing.T, req *TstReq) *TstReply {
		assert.Empty(t, req.Accept)
		output := `{"attributes": [{ "name" : "id"}, {"name": "userName"}]}`
		return &TstReply{Output: output}
	}
	srv := StartTstServer(t, map[string]TstHandler{"GET/scim/Schemas?filter=name+eq+%22User%22": h})
	ctx := NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
	CmdSchema(ctx, "User")
	assert.Empty(t, ctx.Log.ErrString())
	assert.Contains(t, ctx.Log.InfoString(), "Schema for User")
	assert.Contains(t, ctx.Log.InfoString(), "attributes")
	assert.Contains(t, ctx.Log.InfoString(), "name: userName")
}

func NewTestContext(t *testing.T, paths map[string]TstHandler) (*httptest.Server, *HttpContext) {
	srv := StartTstServer(t, paths)
	return srv, NewHttpContext(NewBufferedLogr(), srv.URL, "/", "", false)
}

// Assert context info contains the given string
func AssertOnlyInfoContains(t *testing.T, ctx *HttpContext, expected string) {
	assert.Empty(t, ctx.Log.ErrString(), "Error message should be empty")
	assert.Contains(t, ctx.Log.InfoString(), expected, "INFO log message should contain '"+expected+"'")
}

// Assert context error contains the given string, and info is empty
func AssertOnlyErrorContains(t *testing.T, ctx *HttpContext, expected string) {
	assert.Empty(t, ctx.Log.InfoString(), "Info message should be empty")
	assert.Contains(t, ctx.Log.ErrString(), expected, "ERROR log message should contain '"+expected+"'")
}

// Assert context error contains the given string
func AssertErrorContains(t *testing.T, ctx *HttpContext, expected string) {
	assert.Contains(t, ctx.Log.ErrString(), expected, "ERROR log message should contain '"+expected+"'")
}
