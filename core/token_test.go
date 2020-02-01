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
	"fmt"
	"io/ioutil"
	"net"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/vmware/priam/testaid"
	"github.com/vmware/priam/util"
	. "github.com/vmware/priam/util"
	"gopkg.in/ini.v1"
)

const (
	goodAccessToken = "travolta.was.here"
)

var testTS = TokenService{"/base", "/authorize", "/token", "/login", "salo", "tralfamadore"}

/* in these tests the clientID is "john" and the client secret is "travolta". These are adapted
   from tests written by Fanny, who apparently likes John Travolta.
*/

func TestCanHandleBadCredsGrantReply(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.TokenPath: GoodPathHandler("crap")})
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
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.TokenPath: handler})
	defer srv.Close()
	ti, err := testTS.ClientCredentialsGrant(ctx, "john", "travolta")
	assert.Nil(t, err)
	assert.Equal(t, ti.AccessTokenType, "Bearer")
	assert.Equal(t, ti.AccessToken, goodAccessToken)
}

func TestCanHandleBadUserLoginReply(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.LoginPath: ErrorHandler(0, "crap")})
	defer srv.Close()
	_, err := testTS.LoginSystemUser(ctx, "eliot", "poo-tee-weet")
	assert.NotNil(t, err, "handle bad json reply")
}

func TestCanHandleUserLoginReplyNoToken(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.LoginPath: ErrorHandler(0, `{"crap":"kazak"}`)})
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
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.LoginPath: handler})
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
		assert.Equal(t, TokenCatcherURI, vals.Get("redirect_uri"))
		assert.Equal(t, testTS.CliClientID, vals.Get("client_id"))
		assert.Equal(t, "code", vals.Get("response_type"))
		assert.Equal(t, "kazak", vals.Get("login_hint"))
		state := vals.Get("state")
		assert.NotNil(t, state)
		hc, outp := NewHttpContext(NewBufferedLogr(), TokenCatcherHost, "", "", false), ""
		switch replyType {
		case goodReply:
			vals = url.Values{"code": {authcode}, "state": {state}}
		case badState:
			vals = url.Values{"code": {authcode}, "state": {"bad-state"}}
		case errorReply:
			vals = url.Values{"error": {"server_error"}, "error_description": {"so it goes..."}, "state": {state}}
		}
		err = hc.Request("GET", TokenCatcherPath+"?"+vals.Encode(), nil, &outp)
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
		assert.Equal(t, TokenCatcherURI, vals.Get("redirect_uri"))
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
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.TokenPath: tokenHandler(authcode)})
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
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.TokenPath: tokenHandler("bad-auth-code")})
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
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.TokenPath: tokenHandler(authcode)})
	defer srv.Close()
	_, err := testTS.AuthCodeGrant(ctx, "kazak")
	assert.NotNil(t, err)
	assert.EqualError(t, err, "failed to get authorization code from server. See browser for error message.")
}

func testAuthCodeFailure(t *testing.T, authcode, errmsg string) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"POST" + testTS.BasePath + testTS.TokenPath: tokenHandler(authcode)})
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

// Used http://kjur.github.io/jsjws/tool_jwt.html
// for public and private keys
const aValidPubKey string = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA33TqqLR3eeUmDtHS89qF
3p4MP7Wfqt2Zjj3lZjLjjCGDvwr9cJNlNDiuKboODgUiT4ZdPWbOiMAfDcDzlOxA
04DDnEFGAf+kDQiNSe2ZtqC7bnIc8+KSG/qOGQIVaay4Ucr6ovDkykO5Hxn7OU7s
Jp9TP9H0JH8zMQA6YzijYH9LsupTerrY3U6zyihVEDXXOv08vBHk50BMFJbE9iwF
wnxCsU5+UZUZYw87Uu0n4LPFS9BT8tUIvAfnRXIEWCha3KbFWmdZQZlyrFw0buUE
f0YN3/Q0auBkdbDR/ES2PbgKTJdkjc/rEeM0TxvOUf7HuUNOhrtAVEN1D5uuxE1W
SwIDAQAB
-----END PUBLIC KEY-----
`

const aValidPrivateKey = `-----BEGIN PRIVATE KEY-----
MIIEvAIBADANBgkqhkiG9w0BAQEFAASCBKYwggSiAgEAAoIBAQDfdOqotHd55SYO
0dLz2oXengw/tZ+q3ZmOPeVmMuOMIYO/Cv1wk2U0OK4pug4OBSJPhl09Zs6IwB8N
wPOU7EDTgMOcQUYB/6QNCI1J7Zm2oLtuchzz4pIb+o4ZAhVprLhRyvqi8OTKQ7kf
Gfs5Tuwmn1M/0fQkfzMxADpjOKNgf0uy6lN6utjdTrPKKFUQNdc6/Ty8EeTnQEwU
lsT2LAXCfEKxTn5RlRljDztS7Sfgs8VL0FPy1Qi8B+dFcgRYKFrcpsVaZ1lBmXKs
XDRu5QR/Rg3f9DRq4GR1sNH8RLY9uApMl2SNz+sR4zRPG85R/se5Q06Gu0BUQ3UP
m67ETVZLAgMBAAECggEADjU54mYvHpICXHjc5+JiFqiH8NkUgOG8LL4kwt3DeBp9
bP0+5hSJH8vmzwJkeGG9L79EWG4b/bfxgYdeNX7cFFagmWPRFrlxbd64VRYFawZH
RJt+2cbzMVI6DL8EK4bu5Ux5qTiV44Jw19hoD9nDzCTfPzSTSGrKD3iLPdnREYaI
GDVxcjBv3Tx6rrv3Z2lhHHKhEHb0RRjATcjAVKV9NZhMajJ4l9pqJ3A4IQrCBl95
ux6Xm1oXP0i6aR78cjchsCpcMXdP3WMsvHgTlsZT0RZLFHrvkiNHlPiil4G2/eHk
wvT//CrcbO6SmI/zCtMmypuHJqcr+Xb7GPJoa64WoQKBgQDwrfelf3Rdfo9kaK/b
rBmbu1++qWpYVPTedQy84DK2p3GE7YfKyI+fhbnw5ol3W1jjfvZCmK/p6eZR4jgy
J0KJ76z53T8HoDTF+FTkR55oM3TEM46XzI36RppWP1vgcNHdz3U4DAqkMlAh4lVm
3GiKPGX5JHHe7tWz/uZ55Kk58QKBgQDtrkqdSzWlOjvYD4mq4m8jPgS7v3hiHd+1
OT8S37zdoT8VVzo2T4SF+fBhI2lWYzpQp2sCjLmCwK9k/Gur55H2kTBTwzlQ6WSL
Te9Zj+eoMGklIirA+8YdQHXrO+CCw9BTJAF+c3c3xeUOLXafzyW29bASGfUtA7Ax
QAsR+Rr3+wKBgAwfZxrh6ZWP+17+WuVArOWIMZFj7SRX2yGdWa/lxwgmNPSSFkXj
hkBttujoY8IsSrTivzqpgCrTCjPTpir4iURzWw4W08bpjd7u3C/HX7Y16Uq8ohEJ
T5lslveDJ3iNljSK74eMK7kLg7fBM7YDogxccHJ1IHsvInp3e1pmZxOxAoGAO+bS
TUQ4N/UuQezgkF3TDrnBraO67leDGwRbfiE/U0ghQvqh5DA0QSPVzlWDZc9KUitv
j8vxsR9o1PW9GS0an17GJEYuetLnkShKK3NWOhBBX6d1yP9rVdH6JhgIJEy/g0Su
z7TAFiFc8i7JF8u4QJ05C8bZAMhOLotqftQeVOMCgYAid8aaRvaM2Q8a42Jn6ZTT
5ms6AvNr98sv0StnfmNQ+EYXN0bEk2huSW+w2hN34TYYBTjViQmHbhudwwu8lVjE
ccDmIXsUFbHVK+kTIpWGGchy5cYPs3k9s1nMR2av0Lojtw9WRY76xRXvN8W6R7Eh
wA2ax3+gEEYpGhjM/lO2Lg==
-----END PRIVATE KEY-----`

const aRandomdIdToken string = `eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL3ZpZG0uZXhhbXBsZS5jb20vU0FBUy9hdXRoIiwic3ViIjoiZmFubnlAdmlkbSIsIm5iZiI6MTQ3ODg4NzA4NywiZXhwIjoxNTEwNDIzMDg3LCJpYXQiOjE0Nzg4ODcwODcsImp0aSI6ImlkMTIzNDU2IiwidHlwIjoiSldUIn0.yh7N7SYRBz4Vau9InpWcqSmYHbCUn9Zg8-lR6KaUYwNmGF5a8qzU6hDXMHwsSSE8H7B8-GGEVfdbFUf_xqCi86192waZ_V-9_yn3nqqfscxJDDttZ-TowKeEvY2awxMBJBh6ji1k6XpD52ASvg5ahaskHr8_KWGiGMzO5dhvSaIpFx5pi2H0tq_YCc-lFsTyG4hanSyBz5qM8kQtXGpZwEJAZhUgKJptucKl59jTG9Pi3wFmnW1c-lCDiN3bY2OKBAbs0aNvHz-TLxzUSuzyBGhJZfaWVpSlOzsC2lWA-Q1ldZZWNvPFfjClbwfjdgL_DolXaJQVbdNUVw8cidNc9w`

const anotherPubKey string = `-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCFF0IzO6EC3fblsXOW1arCdv1f
zGvUhhL/v6753ERLYfpzgJBrBvF/WSLwtuM0P83BlA2fNVjSUtTz/JGDf/X4UhdL
HXDAwHyQZnjwCXMvOOSPA/0nwM8qbSsBUACpG9XrGYlRRELIRrvs3s8F90u8etH0
OD59e38OK/yROJncnwIDAQAB
-----END PUBLIC KEY-----
`

const aHmacSignedToken = `eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL3ZpZG0uZXhhbXBsZS5jb20vU0FBUy9hdXRoIiwic3ViIjoiZmFubnlAdmlkbSIsIm5iZiI6MTQ3ODg5MDcyMiwiZXhwIjoxNTEwNDI2NzIyLCJpYXQiOjE0Nzg4OTA3MjIsImp0aSI6ImlkMTIzNDU2IiwidHlwIjoiSldUIn0.X_F4MFLrrgbj1zh60Hcq5q36N6HyH842yraKEM36bIc`

// helper function to generate a signed token
func generateToken(t *testing.T, notBefore time.Time, expiredAt time.Time, issuer string) string {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"iss": issuer,
		"nbf": notBefore.Unix(),
		"exp": expiredAt.Unix(),
		"iat": time.Now().Unix(),
	})

	// Sign and get the complete encoded token as a string using the secret
	privateKey, _ := jwt.ParseRSAPrivateKeyFromPEM([]byte(aValidPrivateKey))
	assert.NotNil(t, privateKey, "private key should be valid")
	tokenString, _ := token.SignedString(privateKey)
	assert.NotNil(t, tokenString, "Token should have been generated")
	return tokenString
}

func NewTestTokenValidationContext(t *testing.T) (*httptest.Server, *util.HttpContext) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET/SAAS/API/1.0/REST/auth/token?attribute=publicKey&format=pem": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Status: 200, Output: aValidPubKey}
		}})
	return srv, ctx
}

func TestCanValidateToken(t *testing.T) {
	srv, ctx := NewTestTokenValidationContext(t)
	defer srv.Close()
	token := generateToken(t, time.Now(), time.Now().AddDate(0, 0, 1), srv.URL+"/SAAS/auth")
	new(TokenService).ValidateIDToken(ctx, token)
	AssertOnlyInfoContains(t, ctx, "ID token is valid")
	AssertOnlyInfoContains(t, ctx, "iss: "+srv.URL+"/SAAS/auth")
}

func TestCannotValidateTokenIfPublicKeyCannotBeRetrieved(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET/SAAS/API/1.0/REST/auth/token?attribute=publicKey&format=pem": ErrorHandler(500, "my favourite")})
	defer srv.Close()
	new(TokenService).ValidateIDToken(ctx, aRandomdIdToken)
	AssertErrorContains(t, ctx, "Could not fetch public key:")
	AssertErrorContains(t, ctx, "my favourite")
}

func TestCannotValidateTokenIfTokenIsEmpty(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{})
	defer srv.Close()
	new(TokenService).ValidateIDToken(ctx, "")
	AssertErrorContains(t, ctx, "No ID token provided.")
}

func TestCannotValidateTokenIfTokenIsJunk(t *testing.T) {
	srv, ctx := NewTestTokenValidationContext(t)
	defer srv.Close()
	new(TokenService).ValidateIDToken(ctx, "abc")
	AssertErrorContains(t, ctx, "Could not parse the token")
}

func TestInvalidTokenIfTokenIsExpired(t *testing.T) {
	srv, ctx := NewTestTokenValidationContext(t)
	defer srv.Close()
	token := generateToken(t, time.Now(), time.Now().AddDate(0, 0, -1), srv.URL+"/SAAS/auth")
	new(TokenService).ValidateIDToken(ctx, token)
	AssertErrorContains(t, ctx, "Token is expired")
}

func TestInvalidTokenIfTokenNotBeforeIsWrong(t *testing.T) {
	srv, ctx := NewTestTokenValidationContext(t)
	defer srv.Close()
	token := generateToken(t, time.Now().AddDate(0, 0, 1), time.Now(), srv.URL+"/SAAS/auth")
	new(TokenService).ValidateIDToken(ctx, token)
	AssertErrorContains(t, ctx, "Token is not active yet")
}

func TestInvalidTokenIfTokenSignatureIsWrong(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET/SAAS/API/1.0/REST/auth/token?attribute=publicKey&format=pem": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Status: 200, Output: anotherPubKey}
		}})
	defer srv.Close()
	token := generateToken(t, time.Now(), time.Now().AddDate(0, 0, 1), srv.URL+"/SAAS/auth")
	new(TokenService).ValidateIDToken(ctx, token)
	AssertErrorContains(t, ctx, "crypto/rsa: verification error")
}

func TestInvalidTokenIfTokenIssuerIsWrong(t *testing.T) {
	srv, ctx := NewTestTokenValidationContext(t)
	defer srv.Close()
	token := generateToken(t, time.Now(), time.Now().AddDate(0, 0, 1), "invalid-issuer")
	new(TokenService).ValidateIDToken(ctx, token)
	AssertErrorContains(t, ctx, "Invalid issuer: 'invalid-issuer'")
}

func TestInvalidTokenIfSigningMethodIsNotRSA256(t *testing.T) {
	srv, ctx := NewTestTokenValidationContext(t)
	defer srv.Close()
	new(TokenService).ValidateIDToken(ctx, aHmacSignedToken)
	AssertErrorContains(t, ctx, "Unexpected signing method: HS256")
}

func awsStsQueryString(role, idToken string) string {
	vals := make(url.Values)
	vals.Set("Action", "AssumeRoleWithWebIdentity")
	vals.Set("DurationSeconds", "7200")
	vals.Set("RoleSessionName", testTS.CliClientID)
	vals.Set("RoleArn", role)
	vals.Set("WebIdentityToken", idToken)
	vals.Set("Version", "2011-06-15")
	return fmt.Sprintf("?%v", vals.Encode())
}

func awsCredFileContents(accessKeyId, accessKey, sessionToken string) string {
	const sampleAwsCliCfg = `[default]
# default section comment here
aws_access_key_id=AKIAIOSFODNN7EXAMPLE
aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
aws_session_token=0987654321

# section comment here
[kazak]
; another comment here
aws_access_key_id=%s
aws_secret_access_key=%s
aws_session_token=%s

# ending section comment
[salo]
aws_access_key_id=AKIAI44QH8DHBEXAMPLE2
aws_secret_access_key=je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY2
aws_session_token=1234567890

`
	return fmt.Sprintf(sampleAwsCliCfg, accessKeyId, accessKey, sessionToken)
}

func awsStsResponse(accessKeyId, accessKey, sessionToken string) string {
	const responseTemplate = `
<AssumeRoleWithWebIdentityResponse xmlns="https://sts.amazonaws.com/doc/2011-06-15/">
  <AssumeRoleWithWebIdentityResult>
    <SubjectFromWebIdentityToken>amzn1.account.AF6RHO7KZU5XRVQJGXK6HB56KR2A</SubjectFromWebIdentityToken>
    <Audience>client.5498841531868486423.1548@apps.example.com</Audience>
    <AssumedRoleUser>
      <Arn>arn:aws:sts::123456789012:assumed-role/FederatedWebIdentityRole/app1</Arn>
      <AssumedRoleId>AROACLKWSDQRAOEXAMPLE:app1</AssumedRoleId>
    </AssumedRoleUser>
    <Credentials>
      <SessionToken>%s</SessionToken>
      <SecretAccessKey>%s</SecretAccessKey>
      <Expiration>2014-10-24T23:00:23Z</Expiration>
      <AccessKeyId>%s</AccessKeyId>
    </Credentials>
    <Provider>www.amazon.com</Provider>
  </AssumeRoleWithWebIdentityResult>
  <ResponseMetadata>
    <RequestId>ad4156e9-bce1-11e2-82e6-6b6efEXAMPLE</RequestId>
  </ResponseMetadata>
</AssumeRoleWithWebIdentityResponse>`
	return fmt.Sprintf(responseTemplate, sessionToken, accessKey, accessKeyId)
}

const (
	goodAwsRole      = "arn:aws:iam::044114111530:role/space-hound"
	goodIdToken      = "kazak.the.hound.of.space"
	goodAwsProfile   = "kazak"
	goodKeyId        = "thisIsAGoodKeyID"
	goodKey          = "this.is.a.good.key"
	goodSessionToken = "good-session-token"
)

func awsStsHandler(accessKeyId, accessKey, sessionToken string) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Output: awsStsResponse(goodKeyId, goodKey, goodSessionToken), ContentType: `application/xml`}
	}
}

func newStsTestContext(t *testing.T) (*httptest.Server, *HttpContext) {
	return NewTestContext(t, map[string]TstHandler{
		"GET/" + awsStsQueryString(goodAwsRole, goodIdToken): awsStsHandler(goodKeyId, goodKey, goodSessionToken)})
}

func TestCanUpdateAWSCredentials(t *testing.T) {
	// handler takes good id token, returns XML creds
	// setup stub aws sts and set stsURL
	srv, ctx := newStsTestContext(t)
	defer srv.Close()

	// setup temp AWS config file
	cfgFile := WriteTempFile(t, awsCredFileContents("1", "2", "3"))
	defer CleanupTempFile(cfgFile)

	// run command
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, cfgFile.Name(), goodAwsProfile)
	AssertOnlyInfoContains(t, ctx, "Successfully updated AWS credentials file")

	// check aws credentials file contents
	contents, err := ioutil.ReadFile(cfgFile.Name())
	require.Nil(t, err)
	assert.Equal(t, awsCredFileContents(goodKeyId, goodKey, goodSessionToken), string(contents))
}

func TestUpdateAWSCredentialsFailsWithoutIDToken(t *testing.T) {
	log, expected := NewBufferedLogr(), "No ID token provided."
	testTS.UpdateAWSCredentials(log, "", goodAwsRole, "https://nonexxistent.example.com", "/tmp/notused", goodAwsProfile)
	assert.Empty(t, log.InfoString(), "Info message should be empty")
	assert.Contains(t, log.ErrString(), expected, "ERROR log message should contain '"+expected+"'")
}

func TestUpdateAWSCredentialsFailsWithSTSError(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET/" + awsStsQueryString(goodAwsRole, goodIdToken): ErrorHandler(500, "traditional error")})
	defer srv.Close()
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, "", goodAwsProfile)
	AssertOnlyErrorContains(t, ctx, "Error getting AWS credentials: 500 Internal Server Error")
}

func TestUpdateAWSCredentialsFailsWithSTSBadReply(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET/" + awsStsQueryString(goodAwsRole, goodIdToken): GoodPathHandler("bad xml<<<<<")})
	defer srv.Close()
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, "", goodAwsProfile)
	AssertOnlyErrorContains(t, ctx, "Error extracting credentials from AWS STS response: XML syntax error")
}

func TestUpdateAWSCredentialsFailsWithBadFile(t *testing.T) {
	srv, ctx := newStsTestContext(t)
	defer srv.Close()
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, os.TempDir(), goodAwsProfile)
	AssertOnlyErrorContains(t, ctx, `Error loading AWS CLI credentials file`)
	AssertOnlyErrorContains(t, ctx, `is a directory`)
}

func TestUpdateAWSCredentialsHandlesWriteFailure(t *testing.T) {
	srv, ctx := newStsTestContext(t)
	defer srv.Close()
	cfgFile := WriteTempFile(t, awsCredFileContents("1", "2", "3"))
	defer CleanupTempFile(cfgFile)
	funcSave := saveCredFile
	saveCredFile = func(f *ini.File, name string) error { return errors.New("could not save cred file") }
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, cfgFile.Name(), goodAwsProfile)
	saveCredFile = funcSave
	AssertOnlyErrorContains(t, ctx, `Could not update AWS credentials file`)
	AssertOnlyErrorContains(t, ctx, "could not save cred file")
}

func TestUpdateAWSCredentialsCantSaveCreds(t *testing.T) {
	srv, ctx := newStsTestContext(t)
	defer srv.Close()
	cfgFile := WriteTempFile(t, awsCredFileContents("1", "2", "3"))
	defer CleanupTempFile(cfgFile)
	funcSave := updateKeyInCredFile
	updateKeyInCredFile = func(f *ini.File, section, key, value string) error {
		return errors.New("could not update value in section")
	}
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, cfgFile.Name(), goodAwsProfile)
	updateKeyInCredFile = funcSave
	AssertOnlyErrorContains(t, ctx, `Error updating credential in section "kazak" of file `)
	AssertOnlyErrorContains(t, ctx, "could not update value in section")
}

func TestUpdateAWSCredentialsCanCreateCredFile(t *testing.T) {
	srv, ctx := newStsTestContext(t)
	defer srv.Close()

	// create tempfile then delete it, then use that file name for new cred file.
	cfgFile := WriteTempFile(t, "")
	CleanupTempFile(cfgFile)
	testTS.UpdateAWSCredentials(ctx.Log, goodIdToken, goodAwsRole, srv.URL, cfgFile.Name(), "roomsford")
	AssertOnlyInfoContains(t, ctx, "Successfully updated AWS credentials file: "+cfgFile.Name())

	// check aws credentials file contents
	contents, err := ioutil.ReadFile(cfgFile.Name())
	require.Nil(t, err)

	assert.Regexp(t, `^\[roomsford\]\n`, string(contents))
	assert.Contains(t, string(contents), "aws_access_key_id=thisIsAGoodKeyID")
	assert.Contains(t, string(contents), "aws_secret_access_key=this.is.a.good.key")
	assert.Contains(t, string(contents), "aws_session_token=good-session-token")
}
