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
	"crypto/rsa"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/dgrijalva/jwt-go"
	"github.com/toqueteos/webbrowser"
	. "github.com/vmware/priam/util"
	"gopkg.in/ini.v1"
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

// Interface to get tokens via OAuth2 grants, system user login API, and validate them.
type TokenGrants interface {
	ClientCredentialsGrant(ctx *HttpContext, clientID, clientSecret string) (TokenInfo, error)
	LoginSystemUser(ctx *HttpContext, user, password string) (TokenInfo, error)
	AuthCodeGrant(ctx *HttpContext, userHint string) (TokenInfo, error)
	ValidateIDToken(ctx *HttpContext, idToken string)
	UpdateAWSCredentials(log *Logr, idToken, role, stsURL, credFile, profile string)
}

type TokenService struct{ BasePath, AuthorizePath, TokenPath, LoginPath, CliClientID, CliClientSecret string }

/* ClientCredsGrant takes a clientID and clientSecret and makes a request for an access token.
   Returns common TokenInfo.
*/
func (ts TokenService) ClientCredentialsGrant(ctx *HttpContext, clientID, clientSecret string) (ti TokenInfo, err error) {
	ctx.BasicAuth(clientID, clientSecret).ContentType("application/x-www-form-urlencoded")
	err = ctx.Request("POST", ts.BasePath+ts.TokenPath, url.Values{"grant_type": {"client_credentials"}}.Encode(), &ti)
	return
}

/* LoginSystemUser takes a username and password and makes a request for an access token.
   This is not an OAuth2 call but uses a vidm specific API and is only valid for users in the
   system directory users. Returns common TokenInfo.
*/
func (ts TokenService) LoginSystemUser(ctx *HttpContext, user, password string) (ti TokenInfo, err error) {
	outp := struct{ SessionToken string }{}
	inp := fmt.Sprintf(`{"username": "%s", "password": "%s", "issueToken": true}`, user, password)
	if err = ctx.ContentType("json").Accept("json").Request("POST", ts.BasePath+ts.LoginPath, inp, &outp); err == nil {
		if token := outp.SessionToken; token == "" {
			err = errors.New("Invalid response: no token in reply from server")
		} else {
			ti.AccessTokenType, ti.AccessToken = "HZN", token
		}
	}
	return
}

const (
	TokenCatcherPort = "8089"
	TokenCatcherPath = "/authcodecatcher"
	TokenCatcherHost = "http://localhost:" + TokenCatcherPort
	TokenCatcherURI  = TokenCatcherHost + TokenCatcherPath
)

var catcherAddress = ""
var authCodeDelivery, authStateDelivery = make(chan string, 1), make(chan string, 1)
var browserLauncher = webbrowser.Open
var openListener = net.Listen
var readRandomBytes = rand.Read

/* GenerateRandomString returns a URL-safe, base64 encoded securely generated random
   string. It will panic if the system's secure random number generator fails.
*/
func GenerateRandomString(randomByteCount int) string {
	b := make([]byte, randomByteCount)
	if _, err := readRandomBytes(b); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(b)
}

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

	state := GenerateRandomString(32)
	if catcherAddress == "" {
		if listener, err := openListener("tcp", ":"+TokenCatcherPort); err != nil {
			return ti, err
		} else {
			http.HandleFunc(TokenCatcherPath, AuthCodeCatcher)
			go func() {
				err := http.Serve(listener, nil)
				ctx.Log.Err("Local http authcode catcher exited: %v\n", err)
			}()
			catcherAddress = listener.Addr().String()
		}
		ctx.Log.Trace("local server listening on: %s\n", catcherAddress)
	}

	authStateDelivery <- state
	vals := url.Values{"response_type": {"code"}, "client_id": {ts.CliClientID},
		"state": {state}, "redirect_uri": {TokenCatcherURI}}
	if userHint != "" {
		vals.Set("login_hint", userHint)
	}
	authUrl := fmt.Sprintf("%s%s?%s", ctx.HostURL, ts.BasePath+ts.AuthorizePath, vals.Encode())
	ctx.Log.Trace("launching browser with %s\n", authUrl)
	if err = browserLauncher(authUrl); err != nil {
		switch {
		case err == webbrowser.ErrNoCandidates,
			err.Error() == fmt.Errorf("webbrowser: tried to open %q, no screen found", authUrl).Error(),
			err.Error() == fmt.Errorf("webbrowser: tried to open %q, but you are running a shell session", authUrl).Error():
			ctx.Log.Info("Please open \n\t%s \n\t\tin your browser\n", authUrl)
		default:
			return TokenInfo{}, err
		}
	}
	if authcode := <-authCodeDelivery; authcode == "" {
		err = errors.New("failed to get authorization code from server. See browser for error message.")
	} else {
		ctx.Log.Trace("caught authcode: %s\n", authcode)
		inp := url.Values{"grant_type": {"authorization_code"}, "code": {authcode},
			"redirect_uri": {TokenCatcherURI}, "client_id": {ts.CliClientID}}.Encode()
		ctx.BasicAuth(ts.CliClientID, ts.CliClientSecret).ContentType("application/x-www-form-urlencoded")
		err = ctx.Request("POST", ts.BasePath+ts.TokenPath, inp, &ti)
	}
	return
}

/* Fetch the public key to validate JWT. Return the key in PEM format or an error if not found. */
func (ts TokenService) GetPublicKeyPEM(ctx *HttpContext) (pemPublicKey *rsa.PublicKey, err error) {
	// @todo - we could cache the public key
	outp := ""
	if err := ctx.Request("GET", "/SAAS/API/1.0/REST/auth/token?attribute=publicKey&format=pem", nil, &outp); err != nil {
		return nil, err
	}
	return jwt.ParseRSAPublicKeyFromPEM([]byte(outp))
}

/* Validate the ID token (locally). */
func (ts TokenService) ValidateIDToken(ctx *HttpContext, idToken string) {
	if idToken == "" {
		ctx.Log.Err("No ID token provided.")
		return
	}

	// Fetch the public key
	publicKey, err := ts.GetPublicKeyPEM(ctx)
	if err != nil {
		ctx.Log.Err(fmt.Sprintf("Could not fetch public key: %v\n", err))
		return
	}

	// Parse takes the token string and a function for looking up the public key
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		ctx.Log.Debug("token claims: %v\n", token.Claims)

		// validate the algorithm we expect
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v, expect 'RSA256'", token.Header["alg"])
		}
		return publicKey, nil
	})

	if token == nil {
		ctx.Log.Err(fmt.Sprintf("Could not parse the token: %v\n", err))
		return
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		// token is valid (which means claims "exp, iat, nbf" have been validated during the parse),
		// so now check issuer is valid
		expectedIssuer := ctx.HostURL + "/SAAS/auth"
		if !token.Claims.(jwt.MapClaims).VerifyIssuer(expectedIssuer, true) {
			ctx.Log.Err(fmt.Sprintf("Invalid issuer: '%s', expected '%s", claims["iss"], expectedIssuer))
		} else {
			ctx.Log.Info("ID token is valid:\n")
			ctx.Log.PP("claims", claims)
		}
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		// give more information on why this is not valid
		if ve.Errors&jwt.ValidationErrorExpired != 0 {
			ctx.Log.Err("Token is expired: %v\n", claims["exp"])
		} else if ve.Errors&jwt.ValidationErrorNotValidYet != 0 {
			ctx.Log.Err("Token is not active yet\n")
		} else {
			ctx.Log.Err("Could not validate the token: %v\n", err)
		}
	}
}

// define cred file handlers so that they can be stubbed for testing
var saveCredFile = func(f *ini.File, fileName string) error { return f.SaveTo(fileName) }
var updateKeyInCredFile = func(f *ini.File, section, key, value string) error {
	_, err := f.Section(section).NewKey(key, value)
	return err
}

// exchange an ID token for AWS credentials and update them in the credFile
func (ts TokenService) UpdateAWSCredentials(log *Logr, idToken, role, stsURL, credFile, profile string) {
	if idToken == "" {
		log.Err("No ID token provided.")
		return
	}

	// set up and make call to aws sts
	actx, vals, outp := NewHttpContext(log, stsURL, "/", "", false), make(url.Values), ""
	vals.Set("Action", "AssumeRoleWithWebIdentity")
	vals.Set("DurationSeconds", "7200")
	vals.Set("RoleSessionName", ts.CliClientID)
	vals.Set("RoleArn", role)
	vals.Set("WebIdentityToken", idToken)
	vals.Set("Version", "2011-06-15")
	if err := actx.Request("GET", fmt.Sprintf("?%v", vals.Encode()), nil, &outp); err != nil {
		log.Err("Error getting AWS credentials: %v\n", err)
		return
	}

	// extract credentials from XML response
	creds := struct {
		SessionToken    string `xml:"AssumeRoleWithWebIdentityResult>Credentials>SessionToken"`
		SecretAccessKey string `xml:"AssumeRoleWithWebIdentityResult>Credentials>SecretAccessKey"`
		AccessKeyId     string `xml:"AssumeRoleWithWebIdentityResult>Credentials>AccessKeyId"`
		Expiration      string `xml:"AssumeRoleWithWebIdentityResult>Credentials>Expiration"`
	}{}
	if err := xml.Unmarshal([]byte(outp), &creds); err != nil {
		log.Err("Error extracting credentials from AWS STS response: %v\n", err)
		return
	}

	log.Debug("Acquired token with expiration: %s\n", creds.Expiration)
	// save credentials in the specified AWS CLI credentials file
	ini.PrettyFormat = false // we're updating someone's aws config file, don't mess it up.
	if awsCfg, err := ini.LooseLoad(credFile); err != nil {
		log.Err("Error loading AWS CLI credentials file \"%s\": %v\n", credFile, err)
	} else {
		for k, v := range map[string]string{"aws_access_key_id": creds.AccessKeyId,
			"aws_secret_access_key": creds.SecretAccessKey, "aws_session_token": creds.SessionToken} {
			if err := updateKeyInCredFile(awsCfg, profile, k, v); err != nil {
				log.Err("Error updating credential in section \"%s\" of file \"%s\": %v", profile, credFile, err)
				return
			}
		}
		if err := saveCredFile(awsCfg, credFile); err != nil {
			log.Err("Could not update AWS credentials file \"%s\": %v\n", credFile, err)
		} else {
			log.Info("Successfully updated AWS credentials file: %s\n", credFile)
		}
	}
}
