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
	"testing"
)

func TestGetSchemaWithNoMediaType(t *testing.T) {
	h := func(t *testing.T, req *tstReq) *tstReply {
		assert.Empty(t, req.accept)
		output := `{"attributes": [{ "name" : "id"}, {"name": "userName"}]}`
		return &tstReply{output: output}
	}
	srv := StartTstServer(t, map[string]tstHandler{"GET/scim/Schemas?filter=name+eq+%22User%22": h})
	ctx := newHttpContext(newBufferedLogr(), srv.URL, "/", "")
	cmdSchema(ctx, "User")
	assert.Empty(t, ctx.log.errString())
	assert.Contains(t, ctx.log.infoString(), "Schema for User")
	assert.Contains(t, ctx.log.infoString(), "attributes")
	assert.Contains(t, ctx.log.infoString(), "name: userName")
}
