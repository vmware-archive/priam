package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	neturl "net/url"
	"os"
	"path/filepath"
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

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func newReqWithFileUpload(key, mediaType string, content []byte, fileName string) (body []byte, contentType string, err error) {
	file, err := os.Open(fileName)
	if err != nil {
		return
	}
	defer file.Close()
	first512 := make([]byte, 512) // first 512 bytes are used to evaluate mime type
	file.Read(first512)
	file.Seek(0, 0)
	buf := &bytes.Buffer{}
	writer, h := multipart.NewWriter(buf), make(textproto.MIMEHeader)

	// TODO: determine media type from file contents
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename="%s"`,
		escapeQuotes(filepath.Base(fileName))))
	h.Set("Content-Type", http.DetectContentType(first512))
	pw, err := writer.CreatePart(h)
	if err == nil {
		_, err = io.Copy(pw, file)
	}
	if err != nil {
		return
	}

	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="blob"`, key))
	h.Set("Content-Type", fullWksMType(mediaType))
	pw, err = writer.CreatePart(h)
	if err == nil {
		_, err = pw.Write(content)
	}
	if err != nil {
		return
	}

	if err = writer.Close(); err == nil {
		contentType, body = writer.FormDataContentType(), buf.Bytes()
	}
	return
}

func httpReq(method, url string, hdrs hdrMap, input, output interface{}) (err error) {
	body, err := toJson(input)
	if err != nil {
		return
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
	if !good[resp.StatusCode] {
		err = fmt.Errorf("Status: %s\n%s\n", resp.Status, toStringWithStyle(lyaml, body))
	}
	return
}

func basicAuth(name, pwd string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(name+":"+pwd))
}

func fullWksMType(shortType string) string {
	return "application/vnd.vmware.horizon.manager." + shortType + "+json"
}

/* takes up to 3 strings to be used as Authorization, Accept, and Content-Type headers.
 * For Accept and Content-Type headers, if they are empty "application/json" is used.
 * For media types that don't start with "application/", the usual workspace prefix
 * is helpfully added.
 */
func InitHdrs(args ...string) hdrMap {
	n, h := [3]string{"Authorization", "Accept", "Content-Type"}, make(hdrMap)
	for i, a := range args {
		if i == 0 || strings.Contains(a, "/") {
			h[n[i]] = a
		} else if a == "" {
			h[n[i]] = "application/json"
		} else if a != "-" {
			h[n[i]] = fullWksMType(a)
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
