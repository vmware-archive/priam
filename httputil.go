package main

import (
	//"golang.org/x/oauth2"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

var sessionToken string

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

func httpReq(method, path string, hdrs http.Header, input string) (output string, err error) {
	tgt, err := curTarget()
	if err != nil {
		return
	}
	url := tgt.Host + "/SAAS/" + path
	req, err := http.NewRequest(method, url, strings.NewReader(input))
	if err != nil {
		return
	}
	if hdrs != nil {
		req.Header = hdrs
	}
	if sessionToken != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", sessionToken)
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

func httpJson(method, path string, hdrs http.Header, input string, output interface{}) (err error) {
	if hdrs == nil {
		hdrs = make(http.Header)
	}
	hdrs.Set("Accept", "application/json")
	body, err := httpReq(method, path, hdrs, input)
	if err != nil {
		return
	}
	return json.Unmarshal([]byte(body), output)
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
	pvals, hdrs := make(url.Values), make(http.Header)
	pvals.Set("grant_type", "client_credentials")
	hdrs.Set("Content-Type", "application/x-www-form-urlencoded")
	hdrs.Set("Authorization", basicAuth(tgt.ClientID, tgt.ClientSecret))
	if err = httpJson("POST", "API/1.0/oauth2/token", hdrs, pvals.Encode(), &tokenInfo); err != nil {
		return
	}
	hdrs.Set("Authorization", tokenInfo.Token_type+" "+tokenInfo.Access_token)
	if err = httpJson("GET", "API/1.0/REST/oauth2/session", hdrs, "", &sessionInfo); err == nil {
		sessionToken = "Bearer " + sessionInfo.SessionToken
	}
	return
}

func getAuthnJson(prefix, path string, mediaType string) {
	if err := getSessionToken(); err != nil {
		log(lerr, "Error getting session token: %v\n", err)
		return
	}
	hdrs := make(http.Header)
	if mediaType == "" {
		hdrs.Set("Accept", "application/json")
	} else if mediaType != "<none>" {
		hdrs.Set("Accept", "application/vnd.vmware.horizon.manager." + mediaType + "+json")
	}
	body, err := httpReq("GET", path, hdrs, "")
	if err != nil {
		log(lerr, "Error: %v\n", err)
	} else {
		ppJson(linfo, prefix, body)
	}
}


