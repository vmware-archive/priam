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
	"net/url"
	"testing"
)

const (
	goodAccessToken = "travolta.was.here"
)

var testTS = TokenService{"/authorize", "/token", "/login", "salo", "tralfamadore"}

/* in these tests the clientID is "john" and the client secret is "travolta". These are adapted
   from tests written by Fanny, who apparently likes John Travolta.
*/

func TestCanHandleBadCredsGrantReply(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: GoodPathHandler("crap")})
	defer srv.Close()
	_, err := testTS.ClientCredentialsGrant(ctx, "john", "travolta")
	assert.NotNil(t, err, "handle bad json reply")
}

func TestCanLoginWithClientCreds(t *testing.T) {
	handler := func(t *testing.T, req *TstReq) *TstReply {
		assert.Equal(t, "Basic am9objp0cmF2b2x0YQ==", req.Authorization)
		assert.Equal(t, "grant_type=client_credentials", req.Input)
		return &TstReply{Output: `{"token_type": "Bearer", "access_token": "` + goodAccessToken + `"}`}
	}
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: handler})
	defer srv.Close()
	ti, err := testTS.ClientCredentialsGrant(ctx, "john", "travolta")
	assert.Nil(t, err)
	assert.Equal(t, ti.AccessTokenType, "Bearer")
	assert.Equal(t, ti.AccessToken, goodAccessToken)
}

func TestCanHandleBadUserLoginReply(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.LoginPath: ErrorHandler(0, "crap")})
	defer srv.Close()
	_, err := testTS.LoginSystemUser(ctx, "eliot", "poo-tee-weet")
	assert.NotNil(t, err, "handle bad json reply")
}

func TestCanHandleUserLoginReplyNoToken(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.LoginPath: ErrorHandler(0, `{"crap":"kazak"}`)})
	defer srv.Close()
	_, err := testTS.LoginSystemUser(ctx, "eliot", "poo-tee-weet")
	assert.EqualError(t, err, "Invalid response: no token in reply from server")
}

func TestSystemUserLogin(t *testing.T) {
	handler := func(t *testing.T, req *TstReq) *TstReply {
		assert.Contains(t, req.Input, `"username": "john"`)
		assert.Contains(t, req.Input, `"password": "travolta"`)
		assert.Contains(t, req.Input, `"issueToken": true`)
		return &TstReply{Output: `{"admin": false, "sessionToken": "` + goodAccessToken + `"}`}
	}
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.LoginPath: handler})
	defer srv.Close()
	ti, err := testTS.LoginSystemUser(ctx, "john", "travolta")
	assert.Nil(t, err)
	assert.Equal(t, ti.AccessTokenType, "HZN")
	assert.Equal(t, ti.AccessToken, goodAccessToken)
}

func simulateBrowser(t *testing.T, authcode string) func(authzURL string) error {
	return func(authzURL string) error {
		purl, err := url.Parse(authzURL)
		assert.Nil(t, err)
		vals := purl.Query()
		assert.Equal(t, catcherHost+catcherPath, vals.Get("redirect_uri"))
		assert.Equal(t, testTS.CliClientID, vals.Get("client_id"))
		assert.Equal(t, "code", vals.Get("response_type"))
		assert.Equal(t, "kazak", vals.Get("login_hint"))
		state := vals.Get("state")
		assert.NotNil(t, state)
		hc, outp := NewHttpContext(NewBufferedLogr(), catcherHost, "", ""), ""
		vals = url.Values{"code": {authcode}, "state": {state}}
		err = hc.Request("GET", catcherPath+"?"+vals.Encode(), nil, &outp)
		assert.Nil(t, err)
		return nil
	}
}

func tokenHandler(authcode string) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		assert.Equal(t, "Basic c2Fsbzp0cmFsZmFtYWRvcmU=", req.Authorization)
		assert.Contains(t, req.Input, "grant_type=authorization_code")
		assert.Contains(t, req.Input, "code="+authcode)
		assert.Contains(t, req.Input, url.Values{"redirect_uri": {catcherHost + catcherPath}}.Encode())
		assert.Contains(t, req.Input, "client_id="+testTS.CliClientID)
		return &TstReply{Output: `{"token_type": "Bearer", "access_token": "` + goodAccessToken + `"}`}
	}
}

func TestAuthCodeGrant(t *testing.T) {
	authcode := "hi-ho"
	browserLauncher = simulateBrowser(t, authcode)
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: tokenHandler(authcode)})
	defer srv.Close()
	ti, err := testTS.AuthCodeGrant(ctx, "kazak")
	assert.Nil(t, err)
	assert.Equal(t, ti.AccessTokenType, "Bearer")
	assert.Equal(t, ti.AccessToken, goodAccessToken)
	assert.Contains(t, ctx.Log.InfoString(), "caught authcode: "+authcode)
}
