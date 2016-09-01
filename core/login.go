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
	"net/url"
	. "github.com/vmware/priam/util"
)

/* ClientCredsGrant takes a clientID and clientSecret and makes a request for an access token.
   The access token is returned in a string prefixed by the token type for use in an http
   authorization header.
*/
func ClientCredentialsGrant(ctx *HttpContext, path, clientID, clientSecret string) (authHeader string, err error) {
	tokenInfo := struct {
		Access_token, Token_type, Refresh_token, Scope string
		Expires_in                                     int
	}{}
	inp := url.Values{"grant_type": {"client_credentials"}}.Encode()
	ctx.BasicAuth(clientID, clientSecret).ContentType("application/x-www-form-urlencoded")
	if err = ctx.Request("POST", path, inp, &tokenInfo); err == nil {
		authHeader = tokenInfo.Token_type + " " + tokenInfo.Access_token
	}
	return
}

/* LoginSystemUser takes a username and password and makes a request for an access token.
   This is not an OAuth2 call but uses a vidm specific API.
   The access token is returned in a string prefixed by the token type, suitable for use
   in an http authorization header.
*/
func LoginSystemUser(ctx *HttpContext, path, user, password string) (authHeader string, err error) {
	tokenInfo := struct{ SessionToken string }{}
	inp := fmt.Sprintf(`{"username": "%s", "password": "%s", "issueToken": true}`, user, password)
	if err = ctx.ContentType("json").Accept("json").Request("POST", path, inp, &tokenInfo); err == nil {
		authHeader = "HZN " + tokenInfo.SessionToken
	}
	return
}
