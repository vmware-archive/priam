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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"os"
	. "github.com/vmware/priam/testaid"
	. "github.com/vmware/priam/util"
	"strings"
	"testing"
)

func appSearchH(filter, output string, status int) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		assert.Equal(t, "catalog.summary.list+json", req.Accept)
		assert.Equal(t, "catalog.search+json", req.ContentType)
		if status != 0 {
			return &TstReply{Status: status}
		} else if req.Input != filter {
			return &TstReply{Output: `{"items":[]}`, ContentType: "catalog.summary.list+json"}
		}
		return &TstReply{Output: output, ContentType: "catalog.summary.list+json"}
	}
}

func appGetH(output string, status int) func(t *testing.T, req *TstReq) *TstReply {
	return func(t *testing.T, req *TstReq) *TstReply {
		return &TstReply{Status: status, Output: output, ContentType: "catalog.saml20+json"}
	}
}

const appSearchFilter = `{"nameFilter":"olaf"}`
const appSearchResult = `{"items": [{ "name" : "olaf", 
		"uuid": "6c48beb6-afb1-44bc-ad7f-980214ee346c", "catalogItemType": "Saml20"}]}`
const appGetPath = "GET/catalogitems/6c48beb6-afb1-44bc-ad7f-980214ee346c"
const appSearchPath = "POST/catalogitems/search?pageSize=10000"
const appGetResults = `{"catalogItemType" : "snowman", "name": "olaf"}`
const appDeletePath = "DELETE/catalogitems/6c48beb6-afb1-44bc-ad7f-980214ee346c"
const appPutPath = "PUT/catalogitems/6c48beb6-afb1-44bc-ad7f-980214ee346c"

var appSearchGetHandlers = map[string]TstHandler{
	appGetPath:    appGetH(appGetResults, 0),
	appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}

func TestAppGet(t *testing.T) {
	srv, ctx := NewTestContext(t, appSearchGetHandlers)
	defer srv.Close()
	new(IDMApplicationService).Display(ctx, "olaf")
	AssertOnlyInfoContains(t, ctx, `name: olaf`)
}

func TestAppGetNotFound(t *testing.T) {
	srv, ctx := NewTestContext(t, appSearchGetHandlers)
	defer srv.Close()
	appGet(ctx, "sven")
	AssertErrorContains(t, ctx, `No app found with name "sven"`)
}

func TestAppGetError(t *testing.T) {
	paths := map[string]TstHandler{
		appGetPath:    ErrorHandler(403, "not found"),
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	appGet(ctx, "olaf")
	AssertErrorContains(t, ctx, `Error getting app info by uuid: 403 Forbidden`)
}

func TestAppGetErrorDuplicateName(t *testing.T) {
	paths := map[string]TstHandler{
		appGetPath: appGetH(appGetResults, 0),
		appSearchPath: appSearchH(appSearchFilter,
			`{"items": [{ "name" : "olaf", "uuid": "1"}, {"name" : "olaf", "uuid": "2"}]}`, 0)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	appGet(ctx, "olaf")
	AssertErrorContains(t, ctx, `Error getting app info by name: Multiple apps with name "olaf"`)
}

func TestAppList(t *testing.T) {
	paths := map[string]TstHandler{
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	new(IDMApplicationService).List(ctx, 0, "olaf")
	AssertOnlyInfoContains(t, ctx, `name: olaf`)
}

func TestAppListError(t *testing.T) {
	paths := map[string]TstHandler{
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 403)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	appList(ctx, 0, "olaf")
	AssertErrorContains(t, ctx, `Error: 403 Forbidden`)
}

func TestAppDelete(t *testing.T) {
	paths := map[string]TstHandler{
		appDeletePath: GoodPathHandler(""),
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	new(IDMApplicationService).Delete(ctx, "olaf")
	AssertOnlyInfoContains(t, ctx, `app olaf deleted`)
}

func TestAppDeleteNotFound(t *testing.T) {
	paths := map[string]TstHandler{
		appDeletePath: GoodPathHandler(""),
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	appDelete(ctx, "sven")
	AssertErrorContains(t, ctx, `No app found with name "sven"`)
}

func TestAppDeleteError(t *testing.T) {
	paths := map[string]TstHandler{
		appDeletePath: ErrorHandler(403, "App not found"),
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	appDelete(ctx, "olaf")
	AssertErrorContains(t, ctx, `Error deleting app olaf from catalog: 403 Forbidden`)
}

const testManifest = `---
applications:
- name: olaf
  memory: 512M
  instances: 1
  path: build/libs/web-application-1.0.0.BUILD-SNAPSHOT.war
  buildpack: https://github.com/cloudfoundry/java-buildpack/archive/master.zip
  env:
    DIEGO_STAGE_BETA: "true"
  workspace:
    packageVersion: '1.0'
    description: Fanny's Demo App for RADIO
    iconFile: %s
    entitleGroup: ALL USERS
    catalogItemType: Saml20
    jsonTester: %s
    attributeMaps: 
      userName: "${user.userName}"
      firstName: "${user.firstName}"
      lastName: "${user.lastName}"
    accessPolicy: %s
    authInfo:
      type: Saml20
      validityTimeSeconds: 200
      nameIdFormat: urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified
      nameId: "${user.userName}"
      configureAs: "manual"
      audience: "https://test.fanny.audience"
      assertionConsumerServiceUrl: "https://test.fanny/a/{domainName}/acs?RelayState=http://mail.google.com/a/{domainName}"
      recipientName: "https://test.fanny/a/{domainName}/acs"
`

// Handler for adding an application, can contain or not an icon
func multipartH(t *testing.T, req *TstReq) *TstReply {
	mediaType, params, err := mime.ParseMediaType(req.ContentType)
	require.Nil(t, err)
	gotImage := false
	expectImage := false
	if strings.HasPrefix(mediaType, "multipart/") {
		expectImage = true
		mr := multipart.NewReader(strings.NewReader(req.Input), params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			require.Nil(t, err)
			slurp, err := ioutil.ReadAll(p)
			require.Nil(t, err)
			switch p.Header.Get("Content-Type") {
			case "catalog.saml20+json":
				assert.Contains(t, string(slurp), `"description":"Fanny's Demo App for RADIO"`)
			case "image/jpeg":
				gotImage = true
			}
		}
	}
	if expectImage {
		assert.True(t, gotImage, "should get an image")
	}
	assert.Equal(t, req.Accept, "catalog.saml20+json")
	return &TstReply{}
}

const groupGetResult = `{                                                                                                                
  "Resources": [
    {
      "displayName": "ALL USERS",
      "id": "40cefa64-61c6-4971-85f1-3eb4dd14ca69",
      "members": [
        {
          "display": "sven",
          "value": "4c7075b2-ce78-45b1-bad1-aa40080e99b8"
        },
        {
          "display": "olaf",
          "value": "ce61a3e8-8bec-49fe-a4bb-bf36e2e680d3"
        }
      ]
    }
  ]
}
`
const accessPolicyResult = `{"items": [{ "name" : "default_access_policy_set", "uuid": "1977-08-11"}]}`

func (jsonMarshalTester) MarshalJSON() ([]byte, error) {
	return nil, errors.New("json marshal error")
}

type appPubEnv struct {
	accessPolicy, jsonError, iconFile string
	appCheckH, appPutH                func(t *testing.T, req *TstReq) *TstReply
}

const noIconFile = "<none>"

func PublishAppTester(t *testing.T, env appPubEnv) *HttpContext {
	const groupPath = "GET/scim/Groups?count=10000&filter=displayName+eq+%22ALL+USERS%22"
	if env.iconFile == "" {
		env.iconFile = "../resources/vin.jpg"
	} else if env.iconFile == noIconFile {
		env.iconFile = ""
	}
	if env.accessPolicy == "" {
		env.accessPolicy = "default_access_policy_set"
	}
	if env.appCheckH == nil {
		env.appCheckH = appSearchH(appSearchFilter, appSearchResult, 0)
	}
	if env.appPutH == nil {
		env.appPutH = GoodPathHandler("")
	}
	tmpFile := WriteTempFile(t, fmt.Sprintf(testManifest, env.iconFile, env.jsonError, env.accessPolicy))
	defer CleanupTempFile(tmpFile)
	paths := map[string]TstHandler{
		"POST/catalogitems": multipartH,
		appSearchPath:       env.appCheckH,
		"GET/accessPolicies": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Output: accessPolicyResult}
		},
		groupPath:                       GoodPathHandler(groupGetResult),
		"POST/entitlements/definitions": GoodPathHandler(""),
		appPutPath:                      env.appPutH,
	}
	srv, ctx := NewTestContext(t, paths)
	defer srv.Close()
	new(IDMApplicationService).Publish(ctx, tmpFile.Name())
	return ctx
}

func TestPublishApp(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{})
	AssertOnlyInfoContains(t, ctx, `App "olaf" added to the catalog`)
}

func TestPublishAppAccessPolicyNameNotFound(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{accessPolicy: "unknown_access_policy_set"})
	AssertErrorContains(t, ctx, `Could not find access policy`)
}

func TestPublishAppAccessPolicyTMI(t *testing.T) {
	// white space matters in this string as it is plugged into a json manifest
	ctx := PublishAppTester(t, appPubEnv{accessPolicy: `default_access_policy_set
    accessPolicySetUuid: 1234565`})
	AssertErrorContains(t, ctx, `Invalid manifest for olaf: both accessPolicy "default_access_policy_set" and AccessPolicySetUuid "1234565" cannot be specified`)
}

func TestPublishAppCheckAppError(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{appCheckH: ErrorHandler(500, "traditional error")})
	AssertErrorContains(t, ctx, `Error checking if app olaf exists: 500 Internal Server Error`)
}

func TestPublishAppAlreadyExists(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{appCheckH: appGetH(appSearchResult, 0)})
	AssertOnlyInfoContains(t, ctx, `App "olaf" updated to the catalog`)
	AssertOnlyInfoContains(t, ctx, `Entitled group "ALL USERS" to app "olaf"`)
}

func TestPublishAppJsonError(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{jsonError: "json error"})
	AssertErrorContains(t, ctx, `Error converting app olaf to JSON`)
}

func TestPublishNewAppNoIconFile(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{iconFile: noIconFile})
	AssertOnlyInfoContains(t, ctx, `App "olaf" added to the catalog`)
	AssertOnlyInfoContains(t, ctx, `Entitled group "ALL USERS" to app "olaf"`)
}

func TestPublishAppThatExistsNoIconFile(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{appCheckH: appGetH(appSearchResult, 0), iconFile: noIconFile})
	AssertOnlyInfoContains(t, ctx, `App "olaf" updated to the catalog`)
	AssertOnlyInfoContains(t, ctx, `Entitled group "ALL USERS" to app "olaf"`)
}

func TestPublishAppNoIconFileError(t *testing.T) {
	ctx := PublishAppTester(t, appPubEnv{appCheckH: appGetH(appSearchResult, 0), iconFile: noIconFile, appPutH: ErrorHandler(500, "traditional error")})
	AssertErrorContains(t, ctx, `Error updating olaf to the catalog: 500 Internal Server Error`)
}

func TestGetAccessPolicyError(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"GET/accessPolicies": ErrorHandler(403, "not found")})
	defer srv.Close()
	assert.Empty(t, accessPolicyId(ctx, "hans"))
	AssertErrorContains(t, ctx, `Error getting access policies: 403 Forbidden`)
}

func TestGetAccessPolicyNotFound(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{"GET/accessPolicies": GoodPathHandler(accessPolicyResult)})
	defer srv.Close()
	assert.Empty(t, accessPolicyId(ctx, "hans"))
	AssertErrorContains(t, ctx, `Could not find access policy uuid`)
}

func TestCheckAppExistsByName(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		appSearchPath: appSearchH("{}", appSearchResult, 0)})
	defer srv.Close()
	id, err := checkAppExists(ctx, "olaf", "")
	assert.Nil(t, err)
	assert.Equal(t, "6c48beb6-afb1-44bc-ad7f-980214ee346c", id)
}

func TestCheckAppExistsByUUID(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		appSearchPath: appSearchH("{}", appSearchResult, 0)})
	defer srv.Close()
	id, err := checkAppExists(ctx, "sven", "6c48beb6-afb1-44bc-ad7f-980214ee346c")
	assert.Nil(t, err)
	assert.Equal(t, "6c48beb6-afb1-44bc-ad7f-980214ee346c", id)
}

func TestPublishAppBadManifest(t *testing.T) {
	_, err := os.Stat("manifest.yaml")
	require.True(t, os.IsNotExist(err), "manifest file must not exist")
	srv, ctx := NewTestContext(t, appSearchGetHandlers)
	defer srv.Close()
	PublishApps(ctx, "")
	AssertErrorContains(t, ctx, `Error getting manifest: open manifest.yaml: no such file or directory`)
}
