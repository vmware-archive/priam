package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	neturl "net/url"
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

func ppJson(lt logType, prefix, input interface{}) {
	var err error
	var outp interface{}
	var inp []byte
	if input != nil {
		switch in := input.(type) {
		case string:
			inp = []byte(in)
		case *string:
			inp = []byte(*in)
		case []byte:
			inp = in
		default:
			outp = input
		}
	}
	if outp == nil {
		if inp == nil || len(inp) == 0 {
			log(lt, "%s is empty.\n", prefix)
			return
		}
		err = json.Unmarshal(inp, &outp)
	}
	if err == nil {
		if out, err := json.MarshalIndent(outp, "", "  "); err == nil {
			log(lt, "%s\n%s\n", prefix, string(out))
			return
		}
	}
	log(lt, "%s:\nCould not parse JSON: %v\nraw:\n%v", prefix, err, input)
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
	ppJson(ltrace, "response body", string(body))
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
		err = errors.New(resp.Status)
	}
	return
}

func basicAuth(name, pwd string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(name+":"+pwd))
}

// returns a string with an access token suitable for an authorization header
// in the form "Bearer xxxxxxx"
func clientCredsGrant(url, clientID, clientSecret string) (authHeader string, err error) {
	tokenInfo := struct {
		Access_token, Token_type, Refresh_token, Scope string
		Expires_in                                     int
	}{}
	pvals := make(neturl.Values)
	pvals.Set("grant_type", "client_credentials")
	body := pvals.Encode()
	hdrs := hdrMap{"Content-Type": "application/x-www-form-urlencoded",
		"Authorization": basicAuth(clientID, clientSecret),
		"Accept":        "application/json"}
	if err = httpReq("POST", url, hdrs, &body, &tokenInfo); err != nil {
		return
	}
	authHeader = tokenInfo.Token_type + " " + tokenInfo.Access_token
	return
}

func InitHdrs(mediaType, authHdr string) hdrMap {
	hdrs := hdrMap{"Authorization": authHdr}
	if mediaType == "" {
		hdrs["Accept"] = "application/json"
	} else if mediaType != "<none>" {
		hdrs["Accept"] = "application/vnd.vmware.horizon.manager." + mediaType + "+json"
	}
	return hdrs
}
