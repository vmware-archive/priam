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

// func TestCanNotLoginWithNoTarget(t *testing.T) {
// 	ctx := runner(newTstCtx(t, " "), "login", "c", "s")
// 	ctx.assertOnlyErrContains("no target set")
// }

// func badLoginReply(t *testing.T, req *TstReq) *TstReply {
// 	assert.NotEmpty(t, req.Input)
// 	return &TstReply{Output: "crap"}
// }

// func TestCanHandleBadUserLoginReply(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.LoginPath: badLoginReply})
// 	defer srv.Close()
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john", "travolta")
// 	ctx.assertOnlyErrContains("invalid")
// }

// func TestCanHandleBadOAuthLoginReply(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.TokenPath: badLoginReply})
// 	defer srv.Close()
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "-c", "john", "travolta")
// 	ctx.assertOnlyErrContains("invalid")
// }

// // in these tests the clientID is "john" and the client secret is "travolta"
// // Adapted from tests written by Fanny, who apparently likes John Travolta
// func tstClientCredGrant(t *testing.T, req *TstReq) *TstReply {
// 	assert.Equal(t, "Basic am9objp0cmF2b2x0YQ==", req.Authorization)
// 	assert.Equal(t, "grant_type=client_credentials", req.Input)
// 	return &TstReply{Output: `{"token_type": "Bearer", "access_token": "` + goodAccessToken + `"}`}
// }

// // Helper function for system login
// func assertSystemLoginSucceeded(t *testing.T, ctx *tstCtx) {
// 	assert.Contains(t, ctx.cfg, "authheader: HZN "+goodAccessToken)
// 	ctx.assertOnlyInfoContains("Access token saved")
// }

// // Helper function for OAuth2 login
// func assertOAuth2LoginSucceeded(t *testing.T, ctx *tstCtx) {
// 	assert.Contains(t, ctx.cfg, "authheader: Bearer "+goodAccessToken)
// 	ctx.assertOnlyInfoContains("Access token saved")
// }

// func TestCanLoginAsOAuthClient(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.TokenPath: tstClientCredGrant})
// 	defer srv.Close()
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "-c", "john", "travolta")
// 	assertOAuth2LoginSucceeded(t, ctx)
// }

// func TestPromptForOauthClient(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.TokenPath: tstClientCredGrant})
// 	defer srv.Close()
// 	consoleInput = strings.NewReader("john")
// 	getRawPassword = func() ([]byte, error) { return []byte("travolta"), nil }
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "-c")
// 	assertOAuth2LoginSucceeded(t, ctx)
// 	ctx.assertOnlyInfoContains("Client ID: ")
// 	ctx.assertOnlyInfoContains("Secret: ")
// }

// // Helper methods to check input login request
// func userLoginHandler() func(t *testing.T, req *TstReq) *TstReply {
// 	return func(t *testing.T, req *TstReq) *TstReply {
// 		assert.Contains(t, req.Input, `"username": "john"`)
// 		assert.Contains(t, req.Input, `"password": "travolta"`)
// 		assert.Contains(t, req.Input, `"issueToken": true`)
// 		return &TstReply{Output: `{"admin": false, "sessionToken": "` + goodAccessToken + `"}`}
// 	}
// }

// func TestCanLoginAsUser(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.LoginPath: userLoginHandler()})
// 	defer srv.Close()
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john", "travolta")
// 	assertSystemLoginSucceeded(t, ctx)
// }

// func TestCanLoginAsUserPromptPassword(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.LoginPath: userLoginHandler()})
// 	defer srv.Close()
// 	getRawPassword = func() ([]byte, error) { return []byte("travolta"), nil }
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john")
// 	assertSystemLoginSucceeded(t, ctx)
// 	ctx.assertOnlyInfoContains("Password: ")
// }

// func TestPromptForSystemUserCreds(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.LoginPath: userLoginHandler()})
// 	defer srv.Close()
// 	consoleInput = strings.NewReader("john")
// 	getRawPassword = func() ([]byte, error) { return []byte("travolta"), nil }
// 	ctx := runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login")
// 	assertSystemLoginSucceeded(t, ctx)
// 	ctx.assertOnlyInfoContains("Password: ")
// 	ctx.assertOnlyInfoContains("Username: ")
// }

// func TestPanicIfCantGetUserName(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.LoginPath: userLoginHandler()})
// 	defer srv.Close()
// 	consoleInput = strings.NewReader("")
// 	getRawPassword = func() ([]byte, error) { return []byte("travolta"), nil }
// 	assert.Panics(t, func() { runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login") })
// }

// func TestPanicIfCantGetPassword(t *testing.T) {
// 	srv := StartTstServer(t, map[string]TstHandler{"POST" + tokenService.LoginPath: userLoginHandler()})
// 	defer srv.Close()
// 	getRawPassword = func() ([]byte, error) { return nil, errors.New("getRawPassword failed") }
// 	assert.Panics(t, func() { runner(newTstCtx(t, tstSrvTgt(srv.URL)), "login", "john") })
// }
