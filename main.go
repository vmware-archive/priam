package main

import (
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"strings"
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

func authHeader() (hdr string) {
	var err error
	if hdr, err = getAuthHeader(); err != nil {
		log(lerr, "Error getting access token: %v\n", err)
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
		err = httpReq("GET", tgtURL(path), InitHdrs(authHdr, mediaType), nil, output)
	}
	return
}

func showAuthnJson(prefix, path string, mediaType string) {
	if authHdr := authHeader(); authHdr != "" {
		var body string
		if err := httpReq("GET", tgtURL(path), InitHdrs(authHdr, mediaType), nil, &body); err != nil {
			log(lerr, "Error: %v\n", err)
		} else {
			logpp(linfo, prefix, body)
		}
	}
}

func cmdSchema(c *cli.Context) {
	args := c.Args()
	if len(args) < 1 {
		log(lerr, "schema type must be specified\nSupported types are User, Group, Role, PasswordState, ServiceProviderConfig")
		return
	}
	vals := make(url.Values)
	vals.Set("filter", fmt.Sprintf("name eq \"%s\"", args[0]))
	path := fmt.Sprintf("jersey/manager/api/scim/Schemas?%v", vals.Encode())
	showAuthnJson("Schema for "+args[0], path, "")
}

func cmdLocalUserStore(c *cli.Context) {
	const desc = "Local User Store configuration"
	const path = "jersey/manager/api/localuserstore"
	const mtype = "local.userstore"
	keyvals := make(map[string]interface{})
	for _, arg := range c.Args() {
		kv := strings.SplitAfterN(arg, "=", 2)
		keyvals[strings.TrimSuffix(kv[0], "=")] = kv[1]
	}
	if len(keyvals) == 0 {
		showAuthnJson(desc, path, mtype)
	} else if authHdr := authHeader(); authHdr != "" {
		var output string
		if err := httpReq("PUT", tgtURL(path), InitHdrs(authHdr, mtype, mtype), keyvals, &output); err != nil {
			log(lerr, "Error: %v\n", err)
		} else {
			logpp(linfo, desc, output)
		}
	}
}

func wks(args []string) (err error) {
	app := cli.NewApp()
	app.Name, app.Usage = "wks", "a utility to publish applications to Workspace"
	app.Email, app.Author, app.Writer = "", "", outW
	app.Action = cli.ShowAppHelp
	app.Before = func(c *cli.Context) (err error) {
		debugMode = c.Bool("debug")
		traceMode = c.Bool("trace")
		if c.Bool("json") {
			logStyleDefault = ljson
		}
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
		cli.BoolFlag{
			Name:  "json, j",
			Usage: "print output in json",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "app",
			Usage: "application publishing commands",
			Subcommands: []cli.Command{
				{
					Name:   "add",
					Usage:  "add applications to the catalog",
					Description: "app add [manifest.yaml]",
					Action: cmdAppAdd,
				},
				{
					Name:   "delete",
					Usage:  "delete an app: delete appID",
					Action: cmdAppDel,
				},
			},
		},
		{
			Name:  "apps",
			Usage: "list all applications in the catalog",
			Action: cmdAppList,
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "count",
					Usage: "maximum apps to list",
				},
				cli.StringFlag{
					Name:  "filter",
					Usage: "app name filter",
				},
			},
		},
		{
			Name:  "health",
			Usage: "check workspace service health",
			Action: func(c *cli.Context) {
				body, err := checkHealth()
				if err != nil {
					log(lerr, "Error on Check Health: %v\n", err)
				} else {
					logpp(linfo, "Health info", body)
				}
			},
		},
		{
			Name:        "localuserstore",
			Usage:       "gets/sets local user store configuration",
			Description: "localuserstore [key=value]...",
			Action:      cmdLocalUserStore,
		},
		{
			Name:        "login",
			Usage:       "currently just saves client_id and client_secret",
			Description: "login client_id [client_secret]",
			Action:      cmdLogin,
		},
		{
			Name:  "policies",
			Usage: "get access policies",
			Action: func(c *cli.Context) {
				showAuthnJson("Access Policies", "jersey/manager/api/accessPolicies", "accesspolicyset.list")
			},
		},
		{
			Name:        "schema",
			Usage:       "get scim schema for given type",
			Description: "schema <type>\nSupported types are User, Group, Role, PasswordState, ServiceProviderConfig",
			Action:      cmdSchema,
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
			Name:  "user",
			Usage: "user account commands",
			Subcommands: []cli.Command{
				{
					Name:   "add",
					Usage:  "create user account: add userName [password]",
					Action: cmdAddUser,
					Flags: []cli.Flag{
						cli.StringFlag{
							Name:  "email",
							Usage: "email of the new user account",
						},
						cli.StringFlag{
							Name:  "family",
							Usage: "family name of the new user account",
						},
						cli.StringFlag{
							Name:  "given",
							Usage: "given name of the new user account",
						},
					},
				},
				{
					Name:   "get",
					Usage:  "display user account: get userName",
					Action: cmdGetUser,
				},
				{
					Name:   "delete",
					Usage:  "delete user account: delete userName",
					Action: cmdDelUser,
				},
				{
					Name:   "password",
					Usage:  "set a user's password: password [password]",
					Action: cmdSetPassword,
				},
				{
					Name:   "bulk",
					Usage:  "bulk load users",
					Action: cmdAddUserBulk,
				},
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
