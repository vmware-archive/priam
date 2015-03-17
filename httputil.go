package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var sessionToken string

type hdrMap map[string]string

func ppHeaders(lt logType, prefix string, hdrs http.Header) {
	log(lt, "%s:\n", prefix)
	for k, av := range hdrs {
		//log(lt, "av: %#v\n", av)
		for _, v := range av {
			log(lt, "  %v: %v\n", k, v)
		}

	}
}

func ppJson(lt logType, prefix, s string) {
	v := map[string]interface{}{}
	err := json.Unmarshal([]byte(s), &v)
	if err == nil {
		if out, err := json.MarshalIndent(v, "", "  "); err == nil {
			log(lt, "%s\n%s\n", prefix, string(out))
			return
		}
	}
	log(lt, "%s:\nCould not parse JSON: %v\nraw:\n%v", prefix, err, s)
}

func httpReq(method, path string, hdrs hdrMap, input string) (output string, err error) {
	tgt, err := curTarget()
	if err != nil {
		return
	}
	url := tgt.Host + "/SAAS/" + path
	req, err := http.NewRequest(method, url, strings.NewReader(input))
	if err != nil {
		return
	}
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	log(ltrace, "%s request to : %v\n", method, url)
	ppHeaders(ltrace, "request headers", req.Header)
	if input != "" {
		log(ltrace, "request body: %v\n", input)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	log(ltrace, "response status: %v\n", resp.Status)
	ppHeaders(ltrace, "response headers", resp.Header)
	//log(ltrace, "response headers: %v\n", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	output = string(body)
	ppJson(ltrace, "response body", output)
	//log(ltrace, "response body: %v\n", output)
	if resp.StatusCode != 200 {
		err = errors.New(resp.Status)
	}
	return
}

func httpJson(method, path string, hdrs hdrMap, input string, output interface{}) (err error) {
	if body, err := httpReq(method, path, hdrs, input); err == nil {
		err = json.Unmarshal([]byte(body), output)
	}
	return
}

func basicAuth(name, pwd string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(name+":"+pwd))
}

func getSessionToken() (err error) {
	tokenInfo := struct {
		Access_token, Token_type, Refresh_token, Scope string
		Expires_in                                     int
	}{}
	sessionInfo := struct {
		Id, SessionToken, Firstname, Lastname string
		Admin                                 bool
	}{}

	tgt, err := curTarget()
	if err != nil {
		return
	}
	pvals := make(url.Values)
	pvals.Set("grant_type", "client_credentials")
	hdrs := hdrMap{"Content-Type": "application/x-www-form-urlencoded",
		"Authorization": basicAuth(tgt.ClientID, tgt.ClientSecret),
		"Accept":        "application/json"}
	if err = httpJson("POST", "API/1.0/oauth2/token", hdrs, pvals.Encode(), &tokenInfo); err != nil {
		return
	}
	hdrs["Authorization"] = tokenInfo.Token_type + " " + tokenInfo.Access_token
	if err = httpJson("GET", "API/1.0/REST/oauth2/session", hdrs, "", &sessionInfo); err == nil {
		sessionToken = "Bearer " + sessionInfo.SessionToken
	}
	return
}

func InitHdrMap(mediaType string) (hdrs hdrMap) {
	hdrs = hdrMap{"Authorization": sessionToken}
	if mediaType == "" {
		hdrs["Accept"] = "application/json"
	} else if mediaType != "<none>" {
		hdrs["Accept"] = "application/vnd.vmware.horizon.manager." + mediaType + "+json"
	}
	return
}

func getAuthnJson(path string, mediaType string, output interface{}) (err error) {
	if err = getSessionToken(); err == nil {
		err = httpJson("GET", path, InitHdrMap(mediaType), "", output)
	}
	return
}

func showAuthnJson(prefix, path string, mediaType string) {
	if err := getSessionToken(); err != nil {
		log(lerr, "Error getting session token: %v\n", err)
		return
	}
	body, err := httpReq("GET", path, InitHdrMap(mediaType), "")
	if err != nil {
		log(lerr, "Error: %v\n", err)
	} else {
		ppJson(linfo, prefix, body)
	}
}
