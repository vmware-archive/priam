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

package testaid

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type TstReq struct {
	Accept, ContentType, Authorization, Input string
}

type TstReply struct {
	Output, ContentType, StatusMsg string
	Status                         int
}

type TstHandler func(t *testing.T, req *TstReq) *TstReply

func stringOrDefault(v, dfault string) string {
	if v != "" {
		return v
	}
	return dfault
}

func StartTstServer(t *testing.T, paths map[string]TstHandler) *httptest.Server {
	return httptest.NewServer(makeTstHttpHandler(t, paths))
}

func StartTstTLSServer(t *testing.T, paths map[string]TstHandler) *httptest.Server {
	return httptest.NewTLSServer(makeTstHttpHandler(t, paths))
}

func makeTstHttpHandler(t *testing.T, paths map[string]TstHandler) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Request URL: '%s'\n", r.Method+r.URL.String())
		if rbody, err := ioutil.ReadAll(r.Body); err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %v", err), 404)
		} else if handler, ok := paths[r.Method+r.URL.String()]; !ok {
			http.Error(w, fmt.Sprintf("unknown path: %v", r.Method+r.URL.String()), 404)
			t.Errorf("unknown path: %v\nregistered paths are: %v", r.Method+r.URL.String(), paths)
		} else {
			reply := handler(t, &TstReq{r.Header.Get("Accept"),
				r.Header.Get("Content-Type"), r.Header.Get("Authorization"),
				string(rbody)})
			if reply.Status != 0 && reply.Status != 200 {
				http.Error(w, reply.StatusMsg, reply.Status)
			}
			w.Header().Set("Content-Type", stringOrDefault(reply.ContentType, "application/json"))
			_, err = io.WriteString(w, reply.Output)
			assert.Nil(t, err)
		}
	}
	return http.HandlerFunc(handler)
}

// Returns an error with the given message
func ErrorHandler(status int, message string) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Status: status, StatusMsg: message}
	}
}

func GoodPathHandler(message string) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: message}
	}
}
