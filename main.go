package main

import (
	//"golang.org/x/oauth2"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

/* target is used to encapsulate everything needed to connect to a workspace
   instance. In addition, Current indicates whether this is the current default
   target.
*/
type target struct {
	Host         string
	ClientID     string
	ClientSecret string
}

type wksApp struct {
	packageVersion string
	description    string
	iconFile       string
	attributeMaps  map[string]string
	accessPolicy   string
	authInfo       map[string]string
}

type manifestApp struct {
	name      string
	memory    string
	instances int
	path      string
	buildpack string
	env       map[string]string
	workspace wksApp
}

type appConfig struct {
	CurrentTarget string
	Targets       map[string]target
}

var appCfg appConfig
var inR io.Reader = os.Stdin
var outW io.Writer = os.Stdout
var errW io.Writer = os.Stderr
var inCfg, outCfg, manifest []byte
var debugMode, traceMode bool
var sessionToken string

type logType int

const (
	linfo logType = iota
	lerr
	ldebug
	ltrace
)

func log(lt logType, format string, args ...interface{}) {
	switch lt {
	case linfo:
		fmt.Fprintf(outW, format, args...)
	case lerr:
		fmt.Fprintf(errW, format, args...)
	case ldebug:
		if debugMode {
			fmt.Fprintf(outW, format, args...)
		}
	case ltrace:
		if traceMode {
			fmt.Fprintf(outW, format, args...)
		}
	}
}

func getFile(filename string) (out []byte, err error) {
	fullname, err := filepath.Abs(filename)
	if err == nil {
		out, err = ioutil.ReadFile(fullname)
	}
	return
}

func putFile(filename string, in []byte) (err error) {
	fullname, err := filepath.Abs(filename)
	if err == nil {
		err = ioutil.WriteFile(fullname, in, 0644)
	}
	return
}

func getAppConfig() (err error) {
	appCfg = appConfig{}
	if err = yaml.Unmarshal(inCfg, &appCfg); err != nil {
		return
	}
	if appCfg.CurrentTarget != "" &&
		appCfg.Targets[appCfg.CurrentTarget] != (target{}) {
		return
	}
	for k := range appCfg.Targets {
		appCfg.CurrentTarget = k
		return
	}
	if appCfg.Targets == nil {
		appCfg.Targets = make(map[string]target)
	}
	return
}

func putAppConfig() (err error) {
	outCfg, err = yaml.Marshal(&appCfg)
	return
}

func curTarget() (tgt target, err error) {
	if appCfg.CurrentTarget == "" {
		return tgt, errors.New("no target set")
	}
	return appCfg.Targets[appCfg.CurrentTarget], nil
}

func httpReq(method, path string, hdrs http.Header, input string) (output string, err error) {
	tgt, err := curTarget()
	if err != nil {
		return
	}
	url := tgt.Host + path
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
	log(ltrace, "%s request to : %v\n", url)
	log(ltrace, "request headers: %v\n", req.Header)
	if input != "" {
		log(ltrace, "request body: %v\n", input)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	log(ltrace, "response status: %v\n", resp.Status)
	log(ltrace, "response headers: %v\n", resp.Header)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	output = string(body)
	log(ltrace, "response Body: %v\n", output)
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

func checkHealth() (string, error) {
	return httpReq("GET", "/SAAS/jersey/manager/api/health", nil, "")
}

/*
func getManifest() (err error) {
	var cfg map[interface{}]interface{}
	if err = yaml.Unmarshal(manifest, cfg); err == nil {
		fmt.Printf("\n\nAppValue: %#v\n\n", cfg["applications"])
		for _, app := range cfg["applications"].([]interface{}) {
			for k, v := range app.(map[interface{}]interface{}) {
				fmt.Printf("key %v, Value: %#v\n\n", k, v)
			}
		}
	}
	return
}
*/

/*
func cmdPush(c *cli.Context) {
	println("push app command")
	getManifest()
}
*/

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
	if err = httpJson("POST", "/SAAS/API/1.0/oauth2/token", hdrs, pvals.Encode(), &tokenInfo); err != nil {
		return
	}
	hdrs.Set("Authorization", tokenInfo.Token_type+" "+tokenInfo.Access_token)
	if err = httpJson("GET", "/SAAS/API/1.0/REST/oauth2/session", hdrs, "", &sessionInfo); err == nil {
		sessionToken = "Bearer " + sessionInfo.SessionToken
	}
	return
}

func cmdHealth(c *cli.Context) {
	body, err := checkHealth()
	if err != nil {
		log(lerr, "Error on Check Health: %v\n", err)
	} else {
		log(linfo, "Body: %v\n", body)
	}
}

func cmdLocalUserStore(c *cli.Context) {
	if err := getSessionToken(); err != nil {
		log(lerr, "Error getting session token: %v\n", err)
		return
	}
	body, err := httpReq("GET", "/SAAS/jersey/manager/api/localuserstore", nil, "")
	if err != nil {
		log(lerr, "Error: %v\n", err)
	} else {
		log(linfo, "Body: %v\n", body)
	}
}

func cmdLogin(c *cli.Context) {
	tgt, err := curTarget()
	if err != nil {
		log(lerr, "Error: %v\n", err)
		return
	}
	a := c.Args()
	if len(a) < 2 {
		log(lerr, "must supply clientID and clientSecret on the command line\n")
		return
	}
	appCfg.Targets[appCfg.CurrentTarget] = target{tgt.Host, a[0], a[1]}
	if err := getSessionToken(); err != nil {
		log(lerr, "Error: %v\n", err)
		return
	}
	putAppConfig()
	log(linfo, "clientID and clientSecret saved\n")
}

func wks(args []string) (err error) {
	app := cli.NewApp()
	app.Name = "wks"
	app.Usage = "a utility to publish applications to Workspace"
	app.Action = cli.ShowAppHelp
	app.Email = ""
	app.Author = ""
	app.Writer = outW

	app.Before = func(c *cli.Context) (err error) {
		debugMode = c.Bool("debug")
		traceMode = c.Bool("trace")
		return
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "print debug output",
		},
		cli.BoolFlag{
			Name:  "trace, t",
			Usage: "print all requests and responses",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "health",
			Usage:     "check workspace service health",
			Action:    cmdHealth,
		},
		{
			Name:        "localuserstore",
			Usage:       "gets local user store configuration",
			Action:      cmdLocalUserStore,
		},
		{
			Name:        "login",
			Usage:       "login -- currently just saved client_id and client_secret",
			Description: "login client_id [client_secret]",
			Action:      cmdLogin,
		},
		{
			Name:        "target",
			Usage:       "set or display the target workspace instance",
			Action:      cmdTarget,
			Description: "wks target [new target URL] [targetName]",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force, f",
					Usage: "force target even if workspace instance not reachable",
				},
			},
		},
		{
			Name:        "targets",
			Usage:       "display all target workspace instances",
			Action:      cmdTargets,
			Description: "wks targets",
		},
		/*
			{
				Name:      "push",
				ShortName: "p",
				Usage:     "push an application",
				Action:    cmdPush,
			},
			{
				Name:      "template",
				ShortName: "r",
				Usage:     "options for task templates",
				Subcommands: []cli.Command{
					{
						Name:  "add",
						Usage: "add a new template",
						Action: func(c *cli.Context) {
							println("new task template: ", c.Args().First())
						},
					},
					{
						Name:  "remove",
						Usage: "remove an existing template",
						Action: func(c *cli.Context) {
							println("removed task template: ", c.Args().First())
						},
					},
				},
			},
		*/
	}

	if err = getAppConfig(); err != nil {
		fmt.Fprintf(errW, "failed to parse configuration: %v", err)
	} else if err = app.Run(args); err != nil {
		fmt.Fprintf(errW, "failed to run app: %v", err)
	}
	return
}

func main() {
	var err error
	var configFile string = filepath.Join(os.Getenv("HOME"), ".wks.yaml")
	inCfg, err = getFile(configFile)
	//println(err.Error())
	if err != nil && !strings.HasSuffix(err.Error(), "no such file or directory") {
		panic(err)
	}
	if err = wks(os.Args); err != nil {
		panic(err)
	}
	if len(outCfg) > 0 {
		if err = putFile(configFile, outCfg); err != nil {
			panic(err)
		}
	}
}
