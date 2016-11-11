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
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/toqueteos/webbrowser"
	. "github.com/vmware/priam/util"
	"io"
	"net"
	"net/http"
	"net/url"
)

/* TokenInfo encapsulates various tokens and information returned by OAuth2 token grants.
   See https://tools.ietf.org/html/rfc6749#section-4.1.4 and
   http://openid.net/specs/openid-connect-core-1_0.html#TokenResponse
*/
type TokenInfo struct {
	AccessTokenType string `json:"token_type,omitempty"`
	AccessToken     string `json:"access_token,omitempty"`
	IDToken         string `json:"id_token,omitempty"`
	RefreshToken    string `json:"refresh_token,omitempty"`
	Scope           string `json:"scope,omitempty"`
	ExpiresIn       int    `json:"expires_in,omitempty"`
}

// Interface to get tokens via OAuth2 grants and system user login API.
type TokenGrants interface {
	ClientCredentialsGrant(ctx *HttpContext, clientID, clientSecret string) (TokenInfo, error)
	LoginSystemUser(ctx *HttpContext, user, password string) (TokenInfo, error)
	AuthCodeGrant(ctx *HttpContext, userHint string) (TokenInfo, error)
}

type TokenService struct{ AuthorizePath, TokenPath, LoginPath, CliClientID, CliClientSecret string }

/* ClientCredsGrant takes a clientID and clientSecret and makes a request for an access token.
   Returns common TokenInfo.
*/
func (ts TokenService) ClientCredentialsGrant(ctx *HttpContext, clientID, clientSecret string) (ti TokenInfo, err error) {
	ctx.BasicAuth(clientID, clientSecret).ContentType("application/x-www-form-urlencoded")
	err = ctx.Request("POST", ts.TokenPath, url.Values{"grant_type": {"client_credentials"}}.Encode(), &ti)
	return
}

/* LoginSystemUser takes a username and password and makes a request for an access token.
   This is not an OAuth2 call but uses a vidm specific API and is only valid for users in the
   system directory users. Returns common TokenInfo.
*/
func (ts TokenService) LoginSystemUser(ctx *HttpContext, user, password string) (ti TokenInfo, err error) {
	outp := struct{ SessionToken string }{}
	inp := fmt.Sprintf(`{"username": "%s", "password": "%s", "issueToken": true}`, user, password)
	if err = ctx.ContentType("json").Accept("json").Request("POST", ts.LoginPath, inp, &outp); err == nil {
		if token := outp.SessionToken; token == "" {
			err = errors.New("Invalid response: no token in reply from server")
		} else {
			ti.AccessTokenType, ti.AccessToken = "HZN", token
		}
	}
	return
}

/* GenerateRandomString returns a URL-safe, base64 encoded securely generated random
   string. It will panic if the system's secure random number generator fails.
*/
func GenerateRandomString(randomByteCount int) string {
	b := make([]byte, randomByteCount)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

var catcherAddress, catcherPort, catcherPath = "", "8888", "/authcodecatcher"
var authCodeDelivery, authStateDelivery = make(chan string, 1), make(chan string, 1)
var browserLauncher = webbrowser.Open
var catcherHost = "http://localhost:" + catcherPort

// AuthCodeCatcher receives oauth2 authorization codes
func AuthCodeCatcher(w http.ResponseWriter, req *http.Request) {
	values := req.URL.Query()
	code, state, errType := values.Get("code"), values.Get("state"), values.Get("error")
	if state != <-authStateDelivery || code != "" && errType != "" || code == "" && errType == "" {
		io.WriteString(w, "Invalid authorization code response from server.\n")
		authCodeDelivery <- ""
	} else if code != "" {
		io.WriteString(w, "Authorization code received. Please close this page.\n")
		authCodeDelivery <- code
	} else {
		io.WriteString(w, fmt.Sprintf("Error: %s\nDescription: %s\n", errType, values.Get("error_description")))
		authCodeDelivery <- ""
	}
}

/* AuthCodeGrant takes a clientID and clientSecret and makes a request for id, access, and
   refresh tokens by launching a browser. Returns TokenInfo or an error.
*/
func (ts TokenService) AuthCodeGrant(ctx *HttpContext, userHint string) (ti TokenInfo, err error) {

	if catcherAddress == "" {
		if listener, err := net.Listen("tcp", ":"+catcherPort); err != nil {
			return ti, err
		} else {
			http.HandleFunc(catcherPath, AuthCodeCatcher)
			go func() {
				err := http.Serve(listener, nil)
				ctx.Log.Err("Local http authcode catcher exited: %v\n", err)
			}()
			catcherAddress = listener.Addr().String()
		}
		ctx.Log.Info("local server listening on: %s\n", catcherAddress)
	}

	state, redirUri := GenerateRandomString(32), catcherHost+catcherPath
	authStateDelivery <- state
	vals := url.Values{"response_type": {"code"}, "client_id": {ts.CliClientID},
		"state": {state}, "redirect_uri": {redirUri}}
	if userHint != "" {
		vals.Set("login_hint", userHint)
	}
	authUrl := fmt.Sprintf("%s%s?%s", ctx.HostURL, ts.AuthorizePath, vals.Encode())
	ctx.Log.Info("launching browser with %s\n", authUrl)
	browserLauncher(authUrl)
	if authcode := <-authCodeDelivery; authcode == "" {
		err = errors.New("failed to get authorization code from server. See browser for error message.")
	} else {
		ctx.Log.Info("caught authcode: %s\n", authcode)
		inp := url.Values{"grant_type": {"authorization_code"}, "code": {authcode},
			"redirect_uri": {redirUri}, "client_id": {ts.CliClientID}}.Encode()
		ctx.BasicAuth(ts.CliClientID, ts.CliClientSecret).ContentType("application/x-www-form-urlencoded")
		err = ctx.Request("POST", ts.TokenPath, inp, &ti)
	}
	return
}
