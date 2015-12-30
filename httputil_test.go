package main

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
		if rbody, err := ioutil.ReadAll(r.Body); err != nil {
			http.Error(w, fmt.Sprintf("error reading request body: %v", err), 404)
		} else if handler, ok := paths[r.Method+r.URL.Path]; !ok {
			http.Error(w, "bad path", 404)
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
