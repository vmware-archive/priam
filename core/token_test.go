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
	"errors"
	"github.com/stretchr/testify/assert"
	. "github.com/vmware/priam/testaid"
	. "github.com/vmware/priam/util"
	"net"
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

type codeReplyType int

const (
	goodReply codeReplyType = iota
	badState
	errorReply
)

func simulateBrowser(t *testing.T, replyType codeReplyType, authcode string) func(authzURL string) error {
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
		switch replyType {
		case goodReply:
			vals = url.Values{"code": {authcode}, "state": {state}}
		case badState:
			vals = url.Values{"code": {authcode}, "state": {"bad-state"}}
		case errorReply:
			vals = url.Values{"error": {"server_error"}, "error_description": {"so it goes..."}, "state": {state}}
		}
		err = hc.Request("GET", catcherPath+"?"+vals.Encode(), nil, &outp)
		assert.Nil(t, err)
		return nil
	}
}

func tokenHandler(authcode string) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		assert.Equal(t, "Basic c2Fsbzp0cmFsZmFtYWRvcmU=", req.Authorization)
		assert.Equal(t, "application/x-www-form-urlencoded", req.ContentType)
		vals, err := url.ParseQuery(req.Input)
		assert.Nil(t, err)
		assert.Equal(t, "authorization_code", vals.Get("grant_type"))
		assert.Equal(t, catcherHost+catcherPath, vals.Get("redirect_uri"))
		assert.Equal(t, testTS.CliClientID, vals.Get("client_id"))
		if vals.Get("code") != authcode {
			return &TstReply{Status: 400, Output: `{"error": "invalid_request", "error_description": "so it goes..."}`}
		}
		return &TstReply{Output: `{"token_type": "Bearer", "access_token": "` + goodAccessToken + `"}`}
	}
}

func TestAuthCodeGrant(t *testing.T) {
	authcode := "hi-ho"
	browserLauncher = simulateBrowser(t, goodReply, authcode)
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: tokenHandler(authcode)})
	defer srv.Close()
	ctx.Log.TraceOn = true
	ti, err := testTS.AuthCodeGrant(ctx, "kazak")
	assert.Nil(t, err)
	assert.Equal(t, "Bearer", ti.AccessTokenType)
	assert.Equal(t, goodAccessToken, ti.AccessToken)
	assert.Contains(t, ctx.Log.InfoString(), "caught authcode: "+authcode)
}

func TestHandleBadAuthCode(t *testing.T) {
	browserLauncher = simulateBrowser(t, goodReply, "hi-ho")
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: tokenHandler("bad-auth-code")})
	defer srv.Close()
	_, err := testTS.AuthCodeGrant(ctx, "kazak")
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "400 Bad Request")
	assert.Contains(t, err.Error(), "invalid_request")
	assert.Contains(t, err.Error(), "so it goes...")
}

func TestHandleBadState(t *testing.T) {
	authcode := "hi-ho"
	browserLauncher = simulateBrowser(t, badState, authcode)
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: tokenHandler(authcode)})
	defer srv.Close()
	_, err := testTS.AuthCodeGrant(ctx, "kazak")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "failed to get authorization code from server. See browser for error message.")
}

func testAuthCodeFailure(t *testing.T, authcode, errmsg string) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.TokenPath: tokenHandler(authcode)})
	defer srv.Close()
	_, err := testTS.AuthCodeGrant(ctx, "kazak")
	assert.EqualError(t, err, errmsg)
}

func TestHandleAuthCodeError(t *testing.T) {
	browserLauncher = simulateBrowser(t, errorReply, "hi-ho")
	testAuthCodeFailure(t, "hi-ho", "failed to get authorization code from server. See browser for error message.")
}

func TestCanHandleFailedBrowserLaunch(t *testing.T) {
	browserLauncher = func(authzURL string) error { return errors.New("no browser on tralfamadore") }
	testAuthCodeFailure(t, "hi-ho", "no browser on tralfamadore")
}

func TestCanHandleFailedListener(t *testing.T) {
	origAddress, origListener := catcherAddress, openListener
	catcherAddress = ""
	openListener = func(proto, addr string) (n net.Listener, err error) { return n, errors.New("can't open listener") }
	testAuthCodeFailure(t, "hi-ho", "can't open listener")
	catcherAddress, openListener = origAddress, origListener
}

func TestPanicOnSecureRandomFailure(t *testing.T) {
	readRandomBytes = func(b []byte) (int, error) { return 0, errors.New("random number generator failed") }
	assert.Panics(t, func() { testAuthCodeFailure(t, "hi-ho", "panic") })
}
