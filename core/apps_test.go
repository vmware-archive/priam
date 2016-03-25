package core

import (
	//"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func appSearchH(filter, output string, status int) func(t *testing.T, req *tstReq) *tstReply {
	return func(t *testing.T, req *tstReq) *tstReply {
		assert.Equal(t, "catalog.summary.list+json", req.accept)
		assert.Equal(t, "catalog.search+json", req.contentType)
		if status != 0 {
			return &tstReply{status: status}
		} else if req.input != filter {
			return &tstReply{output: `{"items":[]}`, contentType: "catalog.summary.list+json"}
		}
		return &tstReply{output: output, contentType: "catalog.summary.list+json"}
	}
}

func appGetH(output string, status int) func(t *testing.T, req *tstReq) *tstReply {
	return func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{status: status, output: output, contentType: "catalog.saml20+json"}
	}
}

const appSearchFilter = `{"nameFilter":"olaf"}`
const appSearchResult = `{"items": [{ "name" : "olaf", 
		"uuid": "6c48beb6-afb1-44bc-ad7f-980214ee346c", "catalogItemType": "Saml20"}]}`
const appGetPath = "GET/catalogitems/6c48beb6-afb1-44bc-ad7f-980214ee346c"
const appSearchPath = "POST/catalogitems/search?pageSize=10000"
const appGetResults = `{"catalogItemType" : "snowman", "name": "olaf"}`
const appDeletePath = "DELETE/catalogitems/6c48beb6-afb1-44bc-ad7f-980214ee346c"

var appSearchGetHandlers = map[string]tstHandler{
	appGetPath:    appGetH(appGetResults, 0),
	appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}

func TestAppGet(t *testing.T) {
	srv, ctx := newTestContext(t, appSearchGetHandlers)
	defer srv.Close()
	appGet(ctx, "olaf")
	assertOnlyInfoContains(t, ctx, `name: olaf`)
}

func TestAppGetNotFound(t *testing.T) {
	srv, ctx := newTestContext(t, appSearchGetHandlers)
	defer srv.Close()
	appGet(ctx, "sven")
	assertErrorContains(t, ctx, `No app found with name "sven"`)
}

func TestAppGetErrorDuplicateName(t *testing.T) {
	paths := map[string]tstHandler{
		appGetPath: appGetH(appGetResults, 0),
		appSearchPath: appSearchH(appSearchFilter,
			`{"items": [{ "name" : "olaf", "uuid": "1"}, {"name" : "olaf", "uuid": "2"}]}`, 0)}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	appGet(ctx, "olaf")
	assertErrorContains(t, ctx, `Error getting app info by name: Multiple apps with name "olaf"`)
}

func TestAppList(t *testing.T) {
	paths := map[string]tstHandler{
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	appList(ctx, 0, "olaf")
	assertOnlyInfoContains(t, ctx, `name: olaf`)
}

func TestAppListError(t *testing.T) {
	paths := map[string]tstHandler{
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 403)}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	appList(ctx, 0, "olaf")
	assertErrorContains(t, ctx, `Error: 403 Forbidden`)
}

func TestAppDelete(t *testing.T) {
	appDelH := func(t *testing.T, req *tstReq) *tstReply { return &tstReply{} }
	paths := map[string]tstHandler{
		appDeletePath: appDelH,
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	appDelete(ctx, "olaf")
	assertOnlyInfoContains(t, ctx, `app olaf deleted`)
}

func TestAppDeleteNotFound(t *testing.T) {
	appDelH := func(t *testing.T, req *tstReq) *tstReply { return &tstReply{} }
	paths := map[string]tstHandler{
		appDeletePath: appDelH,
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	appDelete(ctx, "sven")
	assertErrorContains(t, ctx, `No app found with name "sven"`)
}

func TestAppDeleteError(t *testing.T) {
	paths := map[string]tstHandler{
		appDeletePath: ErrorHandler(403, "App not found"),
		appSearchPath: appSearchH(appSearchFilter, appSearchResult, 0)}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	appDelete(ctx, "olaf")
	assertErrorContains(t, ctx, `Error deleting app olaf from catalog: 403 Forbidden`)
}

const testManifest = `---
applications:
- name: fannys-app
  memory: 512M
  instances: 1
  path: build/libs/web-application-1.0.0.BUILD-SNAPSHOT.war
  buildpack: https://github.com/cloudfoundry/java-buildpack/archive/master.zip
  env:
    DIEGO_STAGE_BETA: "true"
  workspace:
    packageVersion: '1.0'
    description: Fanny's Demo App for RADIO
    iconFile: ../resources/vin.jpg
    entitleGroup: ALL USERS
    catalogItemType: Saml20
    attributeMaps: 
      userName: "${user.userName}"
      firstName: "${user.firstName}"
      lastName: "${user.lastName}"
    accessPolicy: default_access_policy_set
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

func multipartH(t *testing.T, req *tstReq) *tstReply {
	mediaType, params, err := mime.ParseMediaType(req.contentType)
	require.Nil(t, err)
	gotImage := false
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(strings.NewReader(req.input), params["boundary"])
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
				//fmt.Printf("Part %#v:\n%q\n", p.Header.Get("Content-Type"), slurp)
			case "image/jpeg":
				gotImage = true
			}
		}
	}
	assert.True(t, gotImage, "should get an image")
	return &tstReply{}
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

func TestPublishApp(t *testing.T) {
	tmpFile := WriteTempFile(t, testManifest)
	defer CleanupTempFile(tmpFile)
	accessPolicyH := func(t *testing.T, req *tstReq) *tstReply {
		return &tstReply{output: `{"items": [{ "name" : "default_access_policy_set", "uuid": "1977-08-11"}]}`}
	}
	const groupPath = "GET/scim/Groups?count=10000&filter=displayName+eq+%22ALL+USERS%22"
	paths := map[string]tstHandler{
		"POST/catalogitems":             multipartH,
		appSearchPath:                   appSearchH(appSearchFilter, appSearchResult, 0),
		"GET/accessPolicies":            accessPolicyH,
		groupPath:                       GoodPathHandler(groupGetResult),
		"POST/entitlements/definitions": GoodPathHandler(""),
	}
	srv, ctx := newTestContext(t, paths)
	defer srv.Close()
	publishApps(ctx, tmpFile.Name())
	assertOnlyInfoContains(t, ctx, `App "fannys-app" added to the catalog`)
}
