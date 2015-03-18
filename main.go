package main

import (
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"os"
)

// target is used to encapsulate everything needed to connect to a workspace instance.
type target struct {
	Host                   string
	ClientID, ClientSecret string `yaml:",omitempty"`
}

type appConfig struct {
	CurrentTarget string
	Targets       map[string]target
}

var appCfg appConfig
var inCfg, outCfg []byte

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

func getAuthHeader() (hdr string, err error) {
	if tgt, err := curTarget(); err == nil {
		url := tgt.Host + "/SAAS/API/1.0/oauth2/token"
		hdr, err = clientCredsGrant(url, tgt.ClientID, tgt.ClientSecret)
	}
	return
}

func tgtURL(path string) string {
	return appCfg.Targets[appCfg.CurrentTarget].Host + "/SAAS/" + path
}

func checkHealth() (output string, err error) {
	err = httpReq("GET", tgtURL("jersey/manager/api/health"), hdrMap{}, nil, &output)
	return
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
	if _, err := getAuthHeader(); err != nil {
		log(lerr, "Error: %v\n", err)
		return
	}
	putAppConfig()
	log(linfo, "clientID and clientSecret saved\n")
}

func getAuthnJson(path string, mediaType string, output interface{}) (err error) {
	if authHdr, err := getAuthHeader(); err == nil {
		err = httpReq("GET", tgtURL(path), InitHdrMap(mediaType, authHdr), nil, output)
	}
	return
}

func showAuthnJson(prefix, path string, mediaType string) {
	var body, authHdr string
	var err error
	if authHdr, err = getAuthHeader(); err != nil {
		log(lerr, "Error getting access token: %v\n", err)
		return
	}
	if err = httpReq("GET", tgtURL(path), InitHdrMap(mediaType, authHdr), nil, &body); err != nil {
		log(lerr, "Error: %v\n", err)
	} else {
		ppJson(linfo, prefix, body)
	}
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
			Name:  "health",
			Usage: "check workspace service health",
			Action: func(c *cli.Context) {
				body, err := checkHealth()
				if err != nil {
					log(lerr, "Error on Check Health: %v\n", err)
				} else {
					ppJson(linfo, "Health info", body)
				}
			},
		},
		{
			Name:  "catalog",
			Usage: "get catalog items",
			Action: func(c *cli.Context) {
				showAuthnJson("Catalog Items", "API/1.0/REST/admin/catalog", "")
			},
		},
		{
			Name:  "policies",
			Usage: "get access policies",
			Action: func(c *cli.Context) {
				showAuthnJson("Access Policies", "jersey/manager/api/accessPolicies", "accesspolicyset.list")
			},
		},
		{
			Name:   "users",
			Usage:  "get users",
			Action: cmdUsers,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "count",
					Usage: "maximum users to get",
				},
				cli.StringFlag{
					Name:  "filter",
					Usage: "SCIM filter",
				},
			},
		},
		{
			Name:   "user",
			Usage:  "create user account: user userName [password]",
			Action: cmdUser,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "email",
					Usage: "email of the new user account",
				},
				cli.StringFlag{
					Name:  "familyname",
					Usage: "family name of the new user account",
				},
				cli.StringFlag{
					Name:  "givenname",
					Usage: "SCIM filter",
				},
			},
		},
		{
			Name:  "localuserstore",
			Usage: "gets local user store configuration",
			Action: func(c *cli.Context) {
				showAuthnJson("Local User Store configuration",
					"jersey/manager/api/localuserstore", "local.userstore")
			},
		},
		{
			Name:        "login",
			Usage:       "currently just saves client_id and client_secret",
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
		{
			Name:   "pub",
			Usage:  "publish an application",
			Action: cmdPublish,
		},
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
	dir, fname := os.Getenv("HOME"), ".wks.yaml"
	inCfg, err = getFile(dir, fname)
	if err != nil {
		panic(err)
	}
	if err = wks(os.Args); err != nil {
		panic(err)
	}
	if len(outCfg) > 0 {
		if err = putFile(dir, fname, outCfg); err != nil {
			panic(err)
		}
	}
}
