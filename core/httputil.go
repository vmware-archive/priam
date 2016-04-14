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
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	neturl "net/url"
	"os"
	"path/filepath"
	"strings"
)

type HttpContext struct {
	log     *logr
	hostURL string

	/* basePath is a convenience so that many callers can use a short
	 * portion of a path from a common root. If a path for a given request
	 * starts with '/', basePath is ignored. Otherwise it is used to prefix
	 * the given path.
	 */
	basePath string

	/* baseMediaType is a convenience so that many callers can use a short
	 * portion of a set of long media type strings.
	 * For example: "application/vnd.vmware.horizon.manager." + shortType + "+json"
	 */
	baseMediaType string
	headers       map[string]string
	client        http.Client
}

func newHttpContext(log *logr, hostURL, basePath, baseMediaType string) *HttpContext {
	return &HttpContext{log: log, hostURL: hostURL, basePath: basePath,
		baseMediaType: baseMediaType, headers: make(map[string]string)}
}

func (ctx *HttpContext) fullMediaType(shortType string) string {
	if shortType == "" || strings.Contains(shortType, "/") {
		return shortType
	} else if shortType == "json" {
		return "application/json"
	}
	return ctx.baseMediaType + shortType + "+json"
}

func (ctx *HttpContext) header(name, value string) *HttpContext {
	if value == "" {
		delete(ctx.headers, name)
	} else {
		ctx.headers[name] = value
	}
	return ctx
}

func (ctx *HttpContext) accept(s string) *HttpContext {
	return ctx.header("Accept", ctx.fullMediaType(s))
}

func (ctx *HttpContext) contentType(s string) *HttpContext {
	return ctx.header("Content-Type", ctx.fullMediaType(s))
}

func (ctx *HttpContext) authorization(s string) *HttpContext {
	return ctx.header("Authorization", s)
}

func (ctx *HttpContext) traceHeaders(prefix string, hdrs *http.Header) {
	if ctx.log.traceOn {
		ctx.log.trace("%s:\n", prefix)
		for k, av := range *hdrs {
			for _, v := range av {
				ctx.log.trace("  %v: %v\n", k, v)
			}
		}
	}
}

func toJson(input interface{}) (output []byte, err error) {
	switch inp := input.(type) {
	case nil:
	case string:
		output = []byte(inp)
	case []byte:
		output = inp
	case *string:
		if inp != nil {
			output = []byte(*inp)
		}
	case *[]byte:
		if inp != nil {
			output = *inp
		}
	default:
		output, err = json.Marshal(inp)
	}
	return
}

func formatReply(ls logStyle, contentType string, body []byte) string {
	if !strings.HasPrefix(contentType, "application/") || !strings.Contains(contentType, "json") {
		return string(body)
	}
	var parsedBody interface{}
	if err := json.Unmarshal(body, &parsedBody); err != nil {
		return string(body)
	}
	return toStringWithStyle(ls, parsedBody)
}

func (ctx *HttpContext) request(method, path string, input, output interface{}) error {
	body, err := toJson(input)
	if err != nil {
		return err
	}
	url := ctx.hostURL + path
	if !strings.HasPrefix(path, "/") {
		url = ctx.hostURL + ctx.basePath + path
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	for k, v := range ctx.headers {
		req.Header.Set(k, v)
	}
	ctx.log.trace("%s request to : %v\n", method, url)
	ctx.traceHeaders("request headers", &req.Header)
	if input != nil {
		ctx.log.trace("request body: %s\n", body)
	}
	resp, err := ctx.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	ctx.log.trace("response status: %v\n", resp.Status)
	ctx.traceHeaders("response headers", &resp.Header)
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return err
	}
	contentType := resp.Header.Get("Content-Type")
	if ctx.log.traceOn && len(body) > 0 {
		ctx.log.trace("response body:\n%s\n", formatReply(ljson, contentType, body))
	}
	if output != nil {
		switch outp := output.(type) {
		case *string:
			*outp = string(body)
		case []byte:
			outp = body
		default:
			if len(body) > 0 {
				err = json.Unmarshal(body, outp)
			}
		}
	}
	good := map[int]bool{200: true, 201: true, 204: true}
	if !good[resp.StatusCode] {
		err = fmt.Errorf("%s\n%s\n", resp.Status, formatReply(ctx.log.style, contentType, body))
	}
	return err
}

func (ctx *HttpContext) fileUploadRequest(method, path, key, mediaType string, content []byte, fileName string, outp interface{}) error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	first512 := make([]byte, 512) // first 512 bytes are used to evaluate mime type
	file.Read(first512)
	file.Seek(0, 0)
	buf := &bytes.Buffer{}
	writer, h := multipart.NewWriter(buf), make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`,
		escapeQuotes(filepath.Base(fileName))))
	h.Set("Content-Type", http.DetectContentType(first512))
	pw, err := writer.CreatePart(h)
	if err == nil {
		_, err = io.Copy(pw, file)
	}
	if err != nil {
		return err
	}

	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="blob"`, key))
	h.Set("Content-Type", ctx.fullMediaType(mediaType))
	pw, err = writer.CreatePart(h)
	if err == nil {
		_, err = pw.Write(content)
	}
	if err != nil {
		return err
	}

	if err = writer.Close(); err != nil {
		return err
	}
	return ctx.contentType(writer.FormDataContentType()).request(method, path, buf.Bytes(), outp)
}

// sets the ctx.authHeader with basic auth
func (ctx *HttpContext) basicAuth(name, pwd string) *HttpContext {
	return ctx.authorization("Basic " + base64.StdEncoding.EncodeToString([]byte(name+":"+pwd)))
}

// sets the ctx.authHeader with an access token suitable for the
// authorization header in the form "Bearer xxxxxxx"
func (ctx *HttpContext) clientCredsGrant(path, clientID, clientSecret string) (err error) {
	tokenInfo := struct {
		Access_token, Token_type, Refresh_token, Scope string
		Expires_in                                     int
	}{}
	inp := neturl.Values{"grant_type": {"client_credentials"}}.Encode()
	ctx.basicAuth(clientID, clientSecret).contentType("application/x-www-form-urlencoded")
	if err = ctx.request("POST", path, inp, &tokenInfo); err == nil {
		ctx.authorization(tokenInfo.Token_type + " " + tokenInfo.Access_token)
	}
	return
}

func (ctx *HttpContext) getPrintJson(prefix, path, mediaType string) {
	var outp interface{}
	if err := ctx.accept(mediaType).request("GET", path, nil, &outp); err != nil {
		ctx.log.err("Error: %v\n", err)
	} else {
		ctx.log.pp(prefix, outp)
	}
}
