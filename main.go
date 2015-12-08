package main

import (
	"errors"
	"fmt"
	"github.com/codegangsta/cli"
	"net/url"
	"os"
	"path/filepath"
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
var configFileName string

func getAppConfig(fileName string) error {
	if fileName == "" {
		fileName = filepath.Join(os.Getenv("HOME"), ".wks.yaml")
	}
	configFileName, appCfg = fileName, appConfig{}
	if err := getYamlFile(fileName, &appCfg); err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("could not read config file %s, error: %v\n", fileName, err)
		}
		appCfg = appConfig{}
	}
	if appCfg.CurrentTarget != "" &&
		appCfg.Targets[appCfg.CurrentTarget] != (target{}) {
		return nil
	}
	for k := range appCfg.Targets {
		appCfg.CurrentTarget = k
		return nil
	}
	if appCfg.Targets == nil {
		appCfg.Targets = make(map[string]target)
	}
	return nil
}

func putAppConfig() {
	if err := putYamlFile(configFileName, &appCfg); err != nil {
		log(lerr, "could not write config file %s, error: %v\n", configFileName, err)
	}
}

func curTarget() (tgt target, err error) {
	if appCfg.CurrentTarget == "" {
		return tgt, errors.New("no target set")
	}
	return appCfg.Targets[appCfg.CurrentTarget], nil
}

func getAuthHeader() (string, error) {
	if tgt, err := curTarget(); err == nil {
		url := tgt.Host + "/SAAS/API/1.0/oauth2/token"
		return clientCredsGrant(url, tgt.ClientID, tgt.ClientSecret)
	} else {
		return "", err
	}
}

func authHeader() string {
	if hdr, err := getAuthHeader(); err != nil {
		log(lerr, "Error getting access token: %v\n", err)
		return ""
	} else {
		return hdr
	}
}

func InitCmd(c *cli.Context, minArgs, maxArgs int) (args []string, authHdr string) {
	args = c.Args()
	if len(args) < minArgs {
		log(lerr, "at least %d arguments must be specified\n", minArgs)
	} else if len(args) > maxArgs {
		log(lerr, "at most %d arguments can be specified\n", minArgs)
	} else {
		authHdr = authHeader()
		for i := len(args); i < maxArgs; i++ {
			args = append(args, "")
		}
	}
	return
}

func tgtURL(path string) string {
	return appCfg.Targets[appCfg.CurrentTarget].Host + "/SAAS/jersey/manager/api/" + path
}

func checkHealth() (output string, err error) {
	err = httpReq("GET", tgtURL("health"), hdrMap{}, nil, &output)
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

func cmdLocalUserStore(c *cli.Context) {
	const desc = "Local User Store configuration"
	const path = "localuserstore"
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

func cmdSchema(c *cli.Context) {
	args := c.Args()
	if len(args) < 1 {
		log(lerr, "schema type must be specified\nSupported types are User, Group, Role, PasswordState, ServiceProviderConfig\n")
		return
	}
	vals := make(url.Values)
	vals.Set("filter", fmt.Sprintf("name eq \"%s\"", args[0]))
	path := fmt.Sprintf("scim/Schemas?%v", vals.Encode())
	showAuthnJson("Schema for "+args[0], path, "")
}

func cmdBefore(c *cli.Context) (err error) {
	debugMode = c.Bool("debug")
	traceMode = c.Bool("trace")
	verboseMode = c.Bool("verbose")
	if c.Bool("json") {
		logStyleDefault = ljson
	}
	return getAppConfig(c.String("config"))
}

func main() {
	var err error
	if strings.HasSuffix(os.Args[0], "cf-wks") {
		if err = getAppConfig(""); err != nil {
			log(lerr, "could not get app config: %v", err)
		} else {
			cfplugin()
		}
		return
	}
	app := cli.NewApp()
	app.Name, app.Usage = "wks", "a utility to publish applications to Workspace"
	app.Email, app.Author, app.Writer = "", "", os.Stdout
	app.Action = cli.ShowAppHelp
	app.Before = cmdBefore
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "config",
			Usage: "specify alternate configuration file. Default is ~/.wks.yaml",
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "print debug output",
		},
		cli.BoolFlag{
			Name:  "json, j",
			Usage: "print output in json",
		},
		cli.BoolFlag{
			Name:  "trace, t",
			Usage: "print all requests and responses",
		},
		cli.BoolFlag{
			Name:  "verbose, V",
			Usage: "print verbose output",
		},
	}

	pageFlags := []cli.Flag{
		cli.IntFlag{
			Name:  "count",
			Usage: "maximum entries to get",
		},
		cli.StringFlag{
			Name:  "filter",
			Usage: "filter such as 'username eq \"joe\"' for SCIM resources",
		},
	}

	memberFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "delete, d",
			Usage: "delete member",
		},
	}

	userAttrFlags := []cli.Flag{
		cli.StringFlag{
			Name:  "email",
			Usage: "email of the user account",
		},
		cli.StringFlag{
			Name:  "family",
			Usage: "family name of the user account",
		},
		cli.StringFlag{
			Name:  "given",
			Usage: "given name of the user account",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:  "app",
			Usage: "application publishing commands",
			Subcommands: []cli.Command{
				{
					Name:        "add",
					Usage:       "add applications to the catalog",
					Description: "app add [manifest.yaml]",
					Action:      cmdAppAdd,
				},
				{
					Name:   "delete",
					Usage:  "delete an app: delete <name>",
					Action: cmdAppDel,
				},
				{
					Name:   "get",
					Usage:  "get an app: get <name>",
					Action: cmdAppGet,
				},
				{
					Name:   "list",
					Usage:  "list all applications in the catalog",
					Flags:  pageFlags,
					Action: cmdAppList,
				},
			},
		},
		{
			Name:  "entitlement",
			Usage: "commands for entitlements",
			Subcommands: []cli.Command{
				{
					Name:        "get",
					Usage:       "get entitlements for a specific user, app, or group",
					Description: "ent get (group|user|app) <name>",
					Action: func(c *cli.Context) {
						cmdEntitlementGet(c)
					},
				},
			},
		},
		{
			Name:  "group",
			Usage: "commands for groups",
			Subcommands: []cli.Command{
				{
					Name:        "get",
					Usage:       "get a specific group",
					Description: "group get <name>",
					Action: func(c *cli.Context) {
						scimGet(c, "Groups", "displayName")
					},
				},
				{
					Name:  "list",
					Usage: "list all groups",
					Flags: pageFlags,
					Action: func(c *cli.Context) {
						scimList(c, "Groups", "Groups", "displayName", "id",
							"members", "display")
					},
				},
				{
					Name:        "member",
					Usage:       "add or remove users from a group",
					Description: "member <groupname> <username>",
					Flags:       memberFlags,
					Action: func(c *cli.Context) {
						scimMember(c, "Groups", "displayName")
					},
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
				showAuthnJson("Access Policies", "accessPolicies", "accesspolicyset.list")
			},
		},
		{
			Name:  "role",
			Usage: "commands for roles",
			Subcommands: []cli.Command{
				{
					Name:  "get",
					Usage: "get specific SCIM role",
					Action: func(c *cli.Context) {
						scimGet(c, "Roles", "displayName")
					},
				},
				{
					Name:  "list",
					Usage: "list all roles",
					Flags: pageFlags,
					Action: func(c *cli.Context) {
						scimList(c, "Roles")
					},
				},
				{
					Name:        "member",
					Usage:       "add or remove users from a role",
					Description: "member <rolename> <username>",
					Flags:       memberFlags,
					Action: func(c *cli.Context) {
						scimMember(c, "Roles", "displayName")
					},
				},
			},
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
					Flags:  userAttrFlags,
				},
				{
					Name:  "get",
					Usage: "display user account: get userName",
					Action: func(c *cli.Context) {
						scimGet(c, "Users", "userName")
					},
				},
				{
					Name:  "delete",
					Usage: "delete user account: delete userName",
					Action: func(c *cli.Context) {
						scimDelete(c, "Users", "userName")
					},
				},
				{
					Name:  "list",
					Usage: "list user accounts",
					Action: func(c *cli.Context) {
						scimList(c, "Users", "Users", "userName", "id",
							"emails", "display", "roles", "groups", "name",
							"givenName", "familyName", "value")
					},
				},
				{
					Name:   "load",
					Usage:  "bulk load users",
					Action: cmdLoadUsers,
				},
				{
					Name:   "password",
					Usage:  "set a user's password: password [password]",
					Action: cmdSetPassword,
				},
				{
					Name:   "update",
					Usage:  "update user account: update userName",
					Action: cmdUpdateUser,
					Flags:  userAttrFlags,
				},
			},
		},
		{
			Name:        "schema",
			Usage:       "get SCIM schema of specific type",
			Description: "schema <type>\nSupported types are User, Group, Role, PasswordState, ServiceProviderConfig\n",
			Action:      cmdSchema,
		},
	}

	if err = app.Run(os.Args); err != nil {
		log(lerr, "failed to run app: %v", err)
	}
}
