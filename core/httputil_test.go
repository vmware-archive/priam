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
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

type tstReq struct {
	accept, contentType, authorization, input string
}

type tstReply struct {
	output, contentType, statusMsg string
	status                         int
}

type tstHandler func(t *testing.T, req *tstReq) *tstReply

func StartTstServer(t *testing.T, paths map[string]tstHandler) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		//fmt.Printf("Request URL=%s%+v\n", r.Method, r.URL)
		if rbody, err := ioutil.ReadAll(r.Body); err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %v", err), 404)
		} else if handler, ok := paths[r.Method+r.URL.String()]; !ok {
			http.Error(w, fmt.Sprintf("unknown path: %v", r.Method+r.URL.String()), 404)
			t.Errorf("unknown path: %v", r.Method+r.URL.String())
		} else {
			reply := handler(t, &tstReq{r.Header.Get("Accept"),
				r.Header.Get("Content-Type"), r.Header.Get("Authorization"),
				string(rbody)})
			if reply.status != 0 && reply.status != 200 {
				http.Error(w, reply.statusMsg, reply.status)
			} else {
				w.Header().Set("Content-Type", stringOrDefault(reply.contentType, "application/json"))
				_, err = io.WriteString(w, reply.output)
				assert.Nil(t, err)
			}
		}
	}
	return httptest.NewServer(http.HandlerFunc(handler))
}

func newTestContext(t *testing.T, paths map[string]tstHandler) (*httptest.Server, *HttpContext) {
	srv := StartTstServer(t, paths)
	return srv, newHttpContext(newBufferedLogr(), srv.URL, "/", "")
}

func TestHttpGet(t *testing.T) {
	h := func(t *testing.T, req *tstReq) *tstReply {
		assert.Empty(t, req.input)
		return &tstReply{output: "ok", contentType: "text/plain"}
	}
	testpath, output, expected := "/testpath", "", "ok"
	srv := StartTstServer(t, map[string]tstHandler{"GET" + testpath: h})
	ctx := newHttpContext(newLogr(), srv.URL, "", "")
	err := ctx.request("GET", testpath, nil, &output)
	assert.Equal(t, nil, err)
	assert.Equal(t, expected, output)
}

// Assert context info contains the given string
func assertOnlyInfoContains(t *testing.T, ctx *HttpContext, expected string) {
	assert.Empty(t, ctx.log.errString(), "Error message should be empty")
	assert.Contains(t, ctx.log.infoString(), expected, "INFO log message should contain '"+expected+"'")
}

// Assert context error contains the given string
func assertErrorContains(t *testing.T, ctx *HttpContext, expected string) {
	assert.Contains(t, ctx.log.errString(), expected, "ERROR log message should contain '"+expected+"'")
}
