package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	neturl "net/url"
	"strings"
)

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

func httpReq(method, url string, hdrs hdrMap, input, output interface{}) (err error) {
	var body []byte
	if input != nil {
		switch inp := input.(type) {
		case *string:
			body = []byte(*inp)
		case []byte:
			body = inp
		default:
			if body, err = json.Marshal(inp); err != nil {
				return
			}
		}
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	log(ltrace, "%s request to : %v\n", method, url)
	ppHeaders(ltrace, "request headers", req.Header)
	if input != nil {
		log(ltrace, "request body: %s\n", body)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	log(ltrace, "response status: %v\n", resp.Status)
	ppHeaders(ltrace, "response headers", resp.Header)
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		return
	}
	logWithStyle(ltrace, ljson, "response body", string(body))
	if output != nil {
		switch outp := output.(type) {
		case *string:
			*outp = string(body)
		case []byte:
			outp = body
		default:
			if len(body) > 0 {
				err = json.Unmarshal(body, outp)
			}
		}
	}
	good := map[int]bool{200: true, 201: true, 204: true}
	//if !good[resp.StatusCode != 200 && resp.StatusCode != 201 && res.StatusCode != 204 {
	if !good[resp.StatusCode] {
		err = fmt.Errorf("Status: %s\n%s\n", resp.Status, toStringWithStyle(lyaml, body))
	}
	return
}

func basicAuth(name, pwd string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(name+":"+pwd))
}

/* takes up to 3 strings to be used as Authorization, Accept, and Content-Type headers.
 * For Accept and Content-Type headers, if they are empty "application/json" is used.
 * For media types that don't start with "application/", the usual workspace prefix
 * is helpfully added.
 */
func InitHdrs(args ...string) hdrMap {
	n, h := [3]string{"Authorization", "Accept", "Content-Type"}, make(hdrMap)
	for i, a := range args {
		if i == 0 || strings.HasPrefix(a, "application/") {
			h[n[i]] = a
		} else if a == "" {
			h[n[i]] = "application/json"
		} else {
			h[n[i]] = "application/vnd.vmware.horizon.manager." + a + "+json"
		}
	}
	return h
}

// returns a string with an access token suitable for an authorization header
// in the form "Bearer xxxxxxx"
func clientCredsGrant(url, clientID, clientSecret string) (authHeader string, err error) {
	tokenInfo := struct {
		Access_token, Token_type, Refresh_token, Scope string
		Expires_in                                     int
	}{}
	body := neturl.Values{"grant_type": {"client_credentials"}}.Encode()
	hdrs := InitHdrs(basicAuth(clientID, clientSecret), "", "application/x-www-form-urlencoded")
	if err = httpReq("POST", url, hdrs, &body, &tokenInfo); err == nil {
		authHeader = tokenInfo.Token_type + " " + tokenInfo.Access_token
	}
	return
}
