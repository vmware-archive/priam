/*
Copyright (c) 2016, 2018 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cli

import (
	"bufio"
	"fmt"
	"github.com/howeyc/gopass"
	"github.com/urfave/cli"
	. "github.com/vmware/priam/core"
	. "github.com/vmware/priam/util"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	vidmBasePath          = "/jersey/manager/api/"
	vidmBaseMediaType     = "application/vnd.vmware.horizon.manager."
	accessTokenOption     = "accesstoken"
	accessTokenTypeOption = "accesstokentype"
	refreshTokenOption    = "refreshtoken"
	idTokenOption         = "idtoken"
	cliClientSecret       = "not-a-secret"
	defaultAwsCredFile    = ".aws/credentials"
	defaultAwsProfile     = "priam"
	defaultAwsStsEndpoint = "https://sts.amazonaws.com"
)

// Initially a const, but now allowed to be a user-changeable value
var cliClientID = "github.com-vmware-priam"

var cliClientRegistration = map[string]interface{}{"clientId": cliClientID, "secret": cliClientSecret,
	"accessTokenTTL": 60 * 60, "authGrantTypes": "authorization_code refresh_token", "displayUserGrant": false,
	"redirectUri": TokenCatcherURI, "refreshTokenTTL": 60 * 60 * 24 * 30, "scope": "openid user profile email admin"}

var registerDescription = `Registers this application as an OAuth2 client in the target tenant so that
   the login option with authorization code flow can be used. You must be logged
   in with a valid token with admin role. The following options are registered:
      client ID: ` + cliClientID + `
      client secret: ` + cliClientSecret + `
      scope: ` + cliClientRegistration["scope"].(string) + `
      grant types: ` + cliClientRegistration["authGrantTypes"].(string) + `
      redirect URI: ` + cliClientRegistration["redirectUri"].(string) + `
      access token lifetime in seconds: ` + fmt.Sprintf("%v", cliClientRegistration["accessTokenTTL"]) + `
      refresh token lifetime in seconds: ` + fmt.Sprintf("%v", cliClientRegistration["refreshTokenTTL"]) + `
`

// service instances for CLI
var usersService DirectoryService = &SCIMUsersService{}
var groupsService DirectoryService = &SCIMGroupsService{}
var rolesService DirectoryService = &SCIMRolesService{}
var appsService ApplicationService = &IDMApplicationService{}
var templateService OauthResource = AppTemplateService
var clientService OauthResource = OauthClientService
var tokenServiceFactory TokenServiceFactory = &TokenServiceFactoryImpl{}

var getRawPassword = gopass.GetPasswd // called via variable so that tests can provide stub
var consoleInput io.Reader = os.Stdin // will be set to other readers for tests

func getArgOrPassword(log *Logr, prompt, arg string, repeat bool) string {
	getPwd := func(prompt string) string {
		log.Info("%s", prompt)
		if s, err := getRawPassword(); err != nil {
			panic(err)
		} else {
			return string(s)
		}
	}

	if arg != "" {
		return arg
	}
	for {
		if pwd := getPwd(prompt + ": "); !repeat || pwd == getPwd(prompt+" again: ") {
			return pwd
		}
		log.Info(prompt + "s didn't match. Try again.")
	}
}

func getOptionalArg(log *Logr, prompt, arg string) string {
	if arg != "" {
		return arg
	}
	log.Info("%s: ", prompt)
	scanner := bufio.NewScanner(consoleInput)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	return scanner.Text()
}

func InitCtx(cfg *Config, authn bool, insecureSkipVerify *bool) *HttpContext {
	if cfg.CurrentTarget == NoTarget {
		cfg.Log.Err("Error: no target set\n")
		return nil
	}
	basePath := vidmBasePath
	if cfg.IsTenantInHost() {
		basePath = "/SAAS" + vidmBasePath
	}
	skipVerify := cfg.OptionAsBool(InsecureSkipVerifyOption)
	// insecureSkipVerify is not nil when passed as a flag on the command-line and takes precedence over the options
	if insecureSkipVerify != nil {
		skipVerify = *insecureSkipVerify
	}
	ctx := NewHttpContext(cfg.Log, cfg.Option(HostOption), basePath, vidmBaseMediaType, skipVerify)
	if authn {
		if token := cfg.Option(accessTokenOption); token == "" {
			cfg.Log.Err("No access token saved for current target. Please log in.\n")
			return nil
		} else {
			ctx.Authorization(cfg.Option(accessTokenTypeOption) + " " + token)
		}
	}
	return ctx
}

func initArgs(cfg *Config, c *cli.Context, minArgs, maxArgs int, validateArgs func([]string) bool) []string {
	args := c.Args()
	if args == nil {
		args = []string{}
	}
	if len(args) < minArgs {
		cfg.Log.Err("\nInput Error: at least %d arguments must be given\n\n", minArgs)
	} else if maxArgs >= 0 && len(args) > maxArgs {
		cfg.Log.Err("\nInput Error: at most %d arguments can be given\n\n", maxArgs)
	} else {
		for i := len(args); i < maxArgs; i++ {
			args = append(args, "")
		}
		if validateArgs == nil || validateArgs(args) {
			return args
		}
	}
	cli.ShowCommandHelp(c, c.Command.Name)
	return nil
}

func initCmd(cfg *Config, c *cli.Context, minArgs, maxArgs int, authn bool, validateArgs func([]string) bool) (args []string, ctx *HttpContext) {
	if args = initArgs(cfg, c, minArgs, maxArgs, validateArgs); args != nil {
		ctx = InitCtx(cfg, authn, nil)
	}
	return
}

func initUserCmd(cfg *Config, c *cli.Context, getPwd bool) (*BasicUser, *HttpContext) {
	maxArgs := 1
	if getPwd {
		maxArgs = 2
	}
	args := initArgs(cfg, c, 1, maxArgs, nil)
	if args == nil {
		return nil, nil
	}
	user := &BasicUser{Name: args[0], Given: c.String("given"),
		Family: c.String("family"), Email: c.String("email")}
	if getPwd {
		user.Pwd = getArgOrPassword(cfg.Log, "Password", args[1], true)
	}
	return user, InitCtx(cfg, true, nil)
}

func checkTarget(cfg *Config, insecureSkipVerify *bool) bool {
	ctx, output := InitCtx(cfg, false, insecureSkipVerify), ""
	if ctx == nil {
		return false
	}
	if err := ctx.Request("GET", "health", nil, &output); err != nil {
		ctx.Log.Err("Error checking health of %s: %v\n", ctx.HostURL, err)
		return false
	}
	ctx.Log.Debug("health check output:\n%s\n", output)
	if !strings.Contains(output, "allOk") {
		ctx.Log.Err("Reply from %s does not meet health check\n", ctx.HostURL)
		return false
	}
	return true
}

func makeOptionMap(c *cli.Context, flags []cli.Flag, name, value string) map[string]interface{} {
	omap := map[string]interface{}{name: value}
	for _, flag := range flags {
		switch f := flag.(type) {
		case cli.StringFlag:
			omap[f.Name] = c.String(f.Name)
		case cli.BoolFlag:
			omap[f.Name] = c.Bool(f.Name)
		case cli.IntFlag:
			omap[f.Name] = c.Int(f.Name)
		default:
			panic(fmt.Errorf(`option type "%T" is not supported`, flag))
		}
	}
	return omap
}

func cmdWithAuth1Arg(cfg *Config, cmd func(*HttpContext, string)) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
			cmd(ctx, args[0])
		}
		return nil
	}
}

func cmdWithAuth0Arg(cfg *Config, cmd func(*HttpContext)) func(c *cli.Context) error {
	return func(c *cli.Context) error {
		if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
			cmd(ctx)
		}
		return nil
	}
}

// User has requested a custom identity provider client id (login or token commands).  So
// need to update the data structures that was using the default value.
func updateClientID(clientID string) {
	cliClientRegistration["clientId"] = clientID
	strings.Replace(registerDescription, cliClientID, clientID, 1)
	cliClientID = clientID
}

func Priam(args []string, defaultCfgFile string, infoW, errorW io.Writer) {
	var err error
	cfg := &Config{}

	// work around error in cli v1.18 by setting package level ErrWriter since
	// app level ErrWriter is ignored for some deprecation warnings.
	cli.ErrWriter = errorW

	app := cli.NewApp()
	app.Name, app.Usage = filepath.Base(args[0]), "a utility to interact with VMware Identity Manager"
	app.Email, app.Author, app.Writer, app.ErrWriter = "", "", infoW, errorW
	app.Action, app.Version = cli.ShowAppHelp, "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{Name: "config", Usage: "specify config file. Def: " + defaultCfgFile},
		cli.BoolFlag{Name: "debug, d", Usage: "print debug output"},
		cli.BoolFlag{Name: "json, j", Usage: "prefer output in json rather than yaml"},
		cli.BoolFlag{Name: "trace, t", Usage: "print all requests and responses"},
		cli.BoolFlag{Name: "verbose, V", Usage: "print verbose output"},
	}
	app.Before = func(c *cli.Context) (err error) {
		log := &Logr{DebugOn: c.Bool("debug"), TraceOn: c.Bool("trace"),
			Style: LYaml, VerboseOn: c.Bool("verbose"), ErrW: errorW, OutW: infoW}
		if c.Bool("json") {
			log.Style = LJson
		}
		if !cfg.Init(log, StringOrDefault(c.String("config"), defaultCfgFile)) {
			return fmt.Errorf("app initialization failed\n")
		}
		return nil
	}

	pageFlags := []cli.Flag{
		cli.IntFlag{Name: "count", Usage: "maximum entries to get"},
		cli.StringFlag{Name: "filter", Usage: "filter such as 'username eq \"joe\"' for SCIM resources"},
	}

	memberFlags := []cli.Flag{
		cli.BoolFlag{Name: "delete, d", Usage: "delete member"},
	}

	userAttrFlags := []cli.Flag{cli.StringFlag{Name: "email", Usage: "email of the user account"},
		cli.StringFlag{Name: "family", Usage: "family name of the user account"},
		cli.StringFlag{Name: "given", Usage: "given name of the user account"},
	}

	templateFlags := []cli.Flag{
		cli.IntFlag{Name: "accessTokenTTL", Usage: "seconds that the access token is valid", Value: 480},
		cli.StringFlag{Name: "authGrantTypes", Value: "authorization_code"},
		cli.BoolFlag{Name: "displayUserGrant", Usage: "prompt for user consent when registering app instance"},
		cli.IntFlag{Name: "length", Value: 32},
		cli.StringFlag{Name: "redirectUri", Value: "horizonapi://oauth2"},
		cli.IntFlag{Name: "refreshTokenTTL", Usage: "seconds that the refresh token is valid", Value: 2628000},
		cli.StringFlag{Name: "resourceUuid", Value: "00000000-0000-0000-0000-000000000000"},
		cli.StringFlag{Name: "scope", Value: "user profile email"},
		cli.StringFlag{Name: "tokenType", Value: "Bearer"}}

	clientFlags := []cli.Flag{
		cli.IntFlag{Name: "accessTokenTTL", Usage: "seconds that the access token is valid", Value: 480},
		cli.StringFlag{Name: "authGrantTypes", Value: "authorization_code"},
		cli.BoolFlag{Name: "displayUserGrant", Usage: "prompt for user consent"},
		cli.BoolFlag{Name: "inheritanceAllowed"},
		cli.BoolFlag{Name: "internalSystemClient"},
		cli.StringFlag{Name: "redirectUri", Value: "horizonapi://oauth2"},
		cli.IntFlag{Name: "refreshTokenTTL", Usage: "seconds that the refresh token is valid", Value: 2628000},
		cli.StringFlag{Name: "rememberAs"},
		cli.StringFlag{Name: "resourceUuid", Value: "00000000-0000-0000-0000-000000000000"},
		cli.StringFlag{Name: "scope", Value: "user profile email"},
		cli.StringFlag{Name: "secret"},
		cli.StringFlag{Name: "strData"},
		cli.IntFlag{Name: "tokenLength", Value: 32},
		cli.StringFlag{Name: "tokenType", Value: "Bearer"},
	}

	app.Commands = []cli.Command{
		{
			Name: "app", Usage: "application publishing commands",
			Subcommands: []cli.Command{
				{
					Name: "add", Usage: "add applications to the catalog", ArgsUsage: "<manifestYAMLFile>",
					Action: cmdWithAuth1Arg(cfg, appsService.Publish),
				},
				{
					Name: "delete", Usage: "delete an app from the catalog", ArgsUsage: "<appName>",
					Action: cmdWithAuth1Arg(cfg, appsService.Delete),
				},
				{
					Name: "get", Usage: "get information about an app", ArgsUsage: "<appName>",
					Action: cmdWithAuth1Arg(cfg, appsService.Display),
				},
				{
					Name: "list", Usage: "list all applications in the catalog", ArgsUsage: " ",
					Flags: pageFlags,
					Action: func(c *cli.Context) error {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							appsService.List(ctx, c.Int("count"), c.String("filter"))
						}
						return nil
					},
				},
			},
		},
		{
			Name: "client", Usage: "oauth2 client application commands",
			Subcommands: []cli.Command{
				{
					Name: "add", Usage: "create an oauth2 client app", ArgsUsage: "<clientId>",
					Flags: clientFlags,
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							clientService.Add(ctx, args[0],
								makeOptionMap(c, clientFlags, "clientId", args[0]))
						}
						return nil
					},
				},
				{
					Name: "get", Usage: "display oauth2 client app", ArgsUsage: "<name>",
					Action: cmdWithAuth1Arg(cfg, clientService.Get),
				},
				{
					Name: "delete", Usage: "delete oauth2 client app", ArgsUsage: "<name>",
					Action: cmdWithAuth1Arg(cfg, clientService.Delete),
				},
				{
					Name: "list", Usage: "list oauth2 client apps", ArgsUsage: " ",
					Action: cmdWithAuth0Arg(cfg, clientService.List),
				},
				{
					Name: "register", Usage: "register priam as an oauth2 client", ArgsUsage: " ",
					Description: registerDescription,
					Action: cmdWithAuth0Arg(cfg, func(ctx *HttpContext) {
						clientService.Add(ctx, cliClientID, cliClientRegistration)
					}),
				},
			},
		},
		{
			Name: "entitlement", Usage: "commands for entitlements",
			Subcommands: []cli.Command{
				{
					Name: "get", ArgsUsage: "(group|user|app) <name>",
					Usage: "gets entitlements for a specific user, app, or group",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 2, 2, true, func(args []string) bool {
							res := HasString(args[0], []string{"group", "user", "app"})
							if !res {
								cfg.Log.Err("First parameter of 'get' must be user, group or app\n")
							}
							return res
						}); ctx != nil {
							GetEntitlement(ctx, args[0], args[1])
						}
						return nil
					},
				},
			},
		},
		{
			Name: "group", Usage: "commands for groups",
			Subcommands: []cli.Command{
				{
					Name: "get", Usage: "get a specific group", ArgsUsage: "get <groupName>",
					Action: cmdWithAuth1Arg(cfg, groupsService.DisplayEntity),
				},
				{
					Name: "list", Usage: "list all groups", ArgsUsage: " ", Flags: pageFlags,
					Action: func(c *cli.Context) error {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							groupsService.ListEntities(ctx, c.Int("count"), c.String("filter"))
						}
						return nil
					},
				},
				{
					Name: "member", Usage: "add or remove users from a group",
					ArgsUsage: "<groupname> <username>", Flags: memberFlags,
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 2, 2, true, nil); ctx != nil {
							groupsService.UpdateMember(ctx, args[0], args[1], c.Bool("delete"))
						}
						return nil
					},
				},
			},
		},
		{
			Name: "health", Usage: "check workspace service health", ArgsUsage: " ",
			Action: func(c *cli.Context) error {
				if _, ctx := initCmd(cfg, c, 0, 0, false, nil); ctx != nil {
					HealthCheck(ctx)
				}
				return nil
			},
		},
		{
			Name: "localuserstore", Usage: "gets/sets local user store configuration",
			ArgsUsage: "[key=value]...",
			Action: func(c *cli.Context) error {
				if args, ctx := initCmd(cfg, c, 0, -1, true, nil); ctx != nil {
					CmdLocalUserStore(ctx, args)
				}
				return nil
			},
		},
		{
			Name: "login", Usage: "gets an access token as a user or service client",
			ArgsUsage:   "[name] [password]",
			Description: "User will be prompted to enter any arguments not specified on command line",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "authcode, a", Usage: "use browser to authenticate via oauth2 authorization code grant"},
				cli.BoolFlag{Name: "client, c", Usage: "authenticate with oauth2 client ID and secret"},
				cli.StringFlag{Name: "id, i", Usage: "Override client id, default is " + cliClientID},
			},
			Action: func(c *cli.Context) (err error) {
				if a, ctx := initCmd(cfg, c, 0, 2, false, nil); ctx != nil {
					if c.String("id") != "" {
						updateClientID(c.String("id"))
					}
					tokenInfo := TokenInfo{}
					tokenService := tokenServiceFactory.GetTokenService(cfg, cliClientID, cliClientSecret)
					if c.Bool("authcode") {
						if tokenInfo, err = tokenService.AuthCodeGrant(ctx, a[0]); err != nil {
							cfg.Log.Err("Error getting tokens via browser: %v\n", err)
							return nil
						}
					} else {
						promptN, promptP, loginFunc := "Username", "Password", tokenService.LoginSystemUser
						if c.Bool("client") {
							promptN, promptP, loginFunc = "Client ID", "Secret", tokenService.ClientCredentialsGrant
						}
						name := getOptionalArg(cfg.Log, promptN, a[0])
						pwd := getArgOrPassword(cfg.Log, promptP, a[1], false)
						if tokenInfo, err = loginFunc(ctx, name, pwd); err != nil {
							cfg.Log.Err("Error getting access token: %v\n", err)
							return nil
						}
					}
					opts := map[string]interface{}{accessTokenTypeOption: tokenInfo.AccessTokenType,
						accessTokenOption: tokenInfo.AccessToken, refreshTokenOption: tokenInfo.RefreshToken,
						idTokenOption: tokenInfo.IDToken}
					if cfg.WithOptions(opts).Save() {
						cfg.Log.Info("Access token saved\n")
					}
				}
				return nil
			},
		},
		{
			Name: "logout", Usage: "deletes access token from configuration store for current target",
			Action: func(c *cli.Context) error {
				if args := initArgs(cfg, c, 0, 0, nil); args != nil &&
					cfg.WithoutOptions(accessTokenTypeOption, accessTokenOption, refreshTokenOption, idTokenOption).Save() {
					cfg.Log.Info("Access token removed\n")
				}
				return nil
			},
		},
		{
			Name: "policies", Usage: "get access policies", ArgsUsage: " ",
			Action: func(c *cli.Context) error {
				if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
					ctx.GetPrintJson("Access Policies", "accessPolicies", "accesspolicyset.list")
				}
				return nil
			},
		},
		{
			Name: "role", Usage: "commands for roles",
			Subcommands: []cli.Command{
				{
					Name: "get", Usage: "get specific SCIM role", ArgsUsage: "<roleName>",
					Action: cmdWithAuth1Arg(cfg, rolesService.DisplayEntity),
				},
				{
					Name: "list", ArgsUsage: " ", Usage: "list all roles", Flags: pageFlags,
					Action: func(c *cli.Context) error {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							rolesService.ListEntities(ctx, c.Int("count"), c.String("filter"))
						}
						return nil
					},
				},
				{
					Name: "member", Usage: "add or remove users from a role",
					ArgsUsage: "<rolename> <username>", Flags: memberFlags,
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 2, 2, true, nil); ctx != nil {
							rolesService.UpdateMember(ctx, args[0], args[1], c.Bool("delete"))
						}
						return nil
					},
				},
			},
		},
		{
			Name: "schema", Usage: "get SCIM schema of specific type", ArgsUsage: "<type>",
			Description: "Supported types are User, Group, Role, PasswordState, ServiceProviderConfig\n",
			Action:      cmdWithAuth1Arg(cfg, CmdSchema),
		},
		{
			Name: "target", Usage: "set or display the target workspace instance",
			ArgsUsage: "[newTargetURL] [targetName]",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force, f", Usage: "force target -- don't validate URL with health check"},
				cli.BoolFlag{Name: "delete, d", Usage: "delete specified or current target"},
				cli.BoolFlag{Name: "delete-all", Usage: "delete all targets"},
				cli.BoolFlag{Name: "insecure-skip-verify", Usage: "insecure-skip-verify target -- accept self-signed certificate without verifying their identity"},
			},
			Action: func(c *cli.Context) error {
				if args := initArgs(cfg, c, 0, 2, nil); args != nil {
					if c.Bool("delete-all") {
						cfg.Clear()
					} else if c.Bool("delete") {
						cfg.DeleteTarget(args[0], args[1])
					} else if args[0] == "" {
						cfg.PrintTarget("current")
					} else if c.Bool("force") {
						cfg.SetTarget(args[0], args[1], c.Bool("insecure-skip-verify"), nil)
					} else {
						cfg.SetTarget(args[0], args[1], c.Bool("insecure-skip-verify"), checkTarget)
					}
				}
				return nil
			},
		},
		{
			Name: "targets", Usage: "display all targets", ArgsUsage: " ",
			Action: func(c *cli.Context) error {
				if initArgs(cfg, c, 0, 0, nil) != nil {
					cfg.ListTargets()
				}
				return nil
			},
		},
		{
			Name: "template", Usage: "oauth2 application template commands",
			Subcommands: []cli.Command{
				{
					Name: "add", Usage: "create an app template", ArgsUsage: "<appProductId>",
					Flags: templateFlags,
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							templateService.Add(ctx, args[0],
								makeOptionMap(c, templateFlags, "appProductId", args[0]))
						}
						return nil
					},
				},
				{
					Name: "get", Usage: "display app template", ArgsUsage: "<appProductId>",
					Action: cmdWithAuth1Arg(cfg, templateService.Get),
				},
				{
					Name: "delete", Usage: "delete app template", ArgsUsage: "<appProductId>",
					Action: cmdWithAuth1Arg(cfg, templateService.Delete),
				},
				{
					Name: "list", Usage: "list app templates", ArgsUsage: " ",
					Action: cmdWithAuth0Arg(cfg, templateService.List),
				},
			},
		},
		{
			Name: "tenant", Usage: "gets/sets tenant configuration", ArgsUsage: "<tenantName> [key=value]...",
			Action: func(c *cli.Context) error {
				if args, ctx := initCmd(cfg, c, 1, -1, true, nil); ctx != nil {
					CmdTenantConfig(ctx, args[0], args[1:])
				}
				return nil
			},
		},
		{
			Name: "token", Usage: "token operations",
			Subcommands: []cli.Command{
				{
					Name: "validate", Usage: "validate the current ID token (if logged in)", ArgsUsage: " ",
					Action: func(c *cli.Context) error {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							tokenService := tokenServiceFactory.GetTokenService(cfg, cliClientID, cliClientSecret)
							tokenService.ValidateIDToken(ctx, cfg.Option(idTokenOption))
						}
						return nil
					},
				},
				{
					Name: "aws", Usage: "Use ID token to update credentials in the AWS CLI configuration file", ArgsUsage: "<aws-role-arn>",
					Flags: []cli.Flag{
						cli.StringFlag{Name: "credfile, c", Usage: "name of file to store AWS credentials. Default is ~/" + defaultAwsCredFile},
						cli.StringFlag{Name: "profile, p", Usage: "Profile in which to store AWS credentials, Default is \"priam'\"."},
						cli.StringFlag{Name: "id, i", Usage: "Override client id, default is " + cliClientID},
					},
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, false, nil); ctx != nil {
							if c.String("id") != "" {
								updateClientID(c.String("id"))
							}
							tokenService := tokenServiceFactory.GetTokenService(cfg, cliClientID, cliClientSecret)
							tokenService.UpdateAWSCredentials(ctx.Log, cfg.Option(idTokenOption),
								args[0], defaultAwsStsEndpoint,
								StringOrDefault(c.String("credfile"), filepath.Join(os.Getenv("HOME"), defaultAwsCredFile)),
								StringOrDefault(c.String("profile"), defaultAwsProfile))
						}
						return nil
					},
				},
			},
		},
		{
			Name: "user", Usage: "user account commands",
			Subcommands: []cli.Command{
				{
					Name: "add", Usage: "create a user account", ArgsUsage: "<userName> [password]",
					Flags: userAttrFlags,
					Action: func(c *cli.Context) error {
						if user, ctx := initUserCmd(cfg, c, true); ctx != nil {
							usersService.AddEntity(ctx, user)
						}
						return nil
					},
				},
				{
					Name: "get", Usage: "display user account", ArgsUsage: "<userName>",
					Action: cmdWithAuth1Arg(cfg, usersService.DisplayEntity),
				},
				{
					Name: "delete", Usage: "delete user account", ArgsUsage: "<userName>",
					Action: cmdWithAuth1Arg(cfg, usersService.DeleteEntity),
				},
				{
					Name: "list", Usage: "list user accounts", ArgsUsage: " ",
					Flags: pageFlags,
					Action: func(c *cli.Context) error {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							usersService.ListEntities(ctx, c.Int("count"), c.String("filter"))
						}
						return nil
					},
				},
				{
					Name: "load", ArgsUsage: "<fileName>", Usage: "loads yaml file of an array of users.",
					Description: "Example yaml file content:\n---\n- {name: joe, given: joseph, pwd: changeme}\n" +
						"- {name: sue, given: susan, family: jones, email: sue@what.com}\n",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							usersService.LoadEntities(ctx, args[0])
						}
						return nil
					},
				},
				{
					Name: "password", Usage: "set a user's password", ArgsUsage: "<username> [password]",
					Description: "If password is not given as an argument, user will be prompted to enter it",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 2, true, nil); ctx != nil {
							usersService.UpdateEntity(ctx, args[0], &BasicUser{Pwd: getArgOrPassword(cfg.Log, "Password", args[1], true)})
						}
						return nil
					},
				},
				{
					Name: "update", Usage: "update user account", ArgsUsage: "<userName>",
					Flags: userAttrFlags,
					Action: func(c *cli.Context) error {
						if user, ctx := initUserCmd(cfg, c, false); ctx != nil {
							usersService.UpdateEntity(ctx, user.Name, user)
						}
						return nil
					},
				},
			},
		},
	}

	if err = app.Run(args); err != nil {
		fmt.Fprintln(errorW, "failed to run app: ", err)
	}
}
