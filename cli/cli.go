/*
Copyright (c) 2016 VMware, Inc. All Rights Reserved.

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
	"fmt"
	"github.com/howeyc/gopass"
	"github.com/urfave/cli"
	"io"
	"path/filepath"
	. "priam/core"
	. "priam/util"
	"strings"
)

const (
	vidmBasePath      = "/SAAS/jersey/manager/api/"
	vidmBaseMediaType = "application/vnd.vmware.horizon.manager."
	vidmTokenPath     = "/SAAS/API/1.0/oauth2/token"
	vidmLoginPath     = "/SAAS/API/1.0/REST/auth/system/login"
)

// Directory service
var usersService DirectoryService = &SCIMUsersService{}
var groupsService DirectoryService = &SCIMGroupsService{}
var rolesService DirectoryService = &SCIMRolesService{}
var appsService ApplicationService = &IDMApplicationService{}

// called via variable so that tests can provide stub
var getRawPassword = gopass.GetPasswd

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

func InitCtx(cfg *Config, authn bool) *HttpContext {
	if cfg.CurrentTarget == NoTarget {
		cfg.Log.Err("Error: no target set\n")
		return nil
	}
	tgt := cfg.Targets[cfg.CurrentTarget]
	ctx := NewHttpContext(cfg.Log, tgt.Host, vidmBasePath, vidmBaseMediaType)
	if authn {
		if hdr := cfg.Targets[cfg.CurrentTarget].AuthHeader; hdr == "" {
			cfg.Log.Err("No access token saved for current target. Please log in.\n")
			return nil
		} else {
			ctx.Authorization(hdr)
		}
	}
	return ctx
}

func initArgs(cfg *Config, c *cli.Context, minArgs, maxArgs int, validateArgs func([]string) bool) []string {
	args := c.Args()
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
		ctx = InitCtx(cfg, authn)
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
	return user, InitCtx(cfg, true)
}

func checkTarget(cfg *Config) bool {
	ctx, output := InitCtx(cfg, false), ""
	if ctx == nil {
		return false
	}
	if err := ctx.Request("GET", "health", nil, &output); err != nil {
		ctx.Log.Err("Error checking health of %s: \n", ctx.HostURL)
		return false
	}
	ctx.Log.Debug("health check output:\n%s\n", output)
	if !strings.Contains(output, "allOk") {
		ctx.Log.Err("Reply from %s does not meet health check\n", ctx.HostURL)
		return false
	}
	return true
}

func Priam(args []string, defaultCfgFile string, infoW, errorW io.Writer) {
	var err error
	var cfg *Config
	cli.HelpFlag.Usage = "show help for given command or subcommand"

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
		fileName := c.String("config")
		if fileName == "" {
			fileName = defaultCfgFile
		}
		if cfg = NewConfig(log, fileName); cfg == nil {
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

	app.Commands = []cli.Command{
		{
			Name: "app", Usage: "application publishing commands",
			Subcommands: []cli.Command{
				{
					Name: "add", Usage: "add applications to the catalog", ArgsUsage: "[./manifest.yaml]",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 0, 1, true, nil); ctx != nil {
							appsService.Publish(ctx, args[0])
						}
						return nil
					},
				},
				{
					Name: "delete", Usage: "delete an app from the catalog", ArgsUsage: "<appName>",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							appsService.Delete(ctx, args[0])
						}
						return nil
					},
				},
				{
					Name: "get", Usage: "get information about an app", ArgsUsage: "<appName>",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							appsService.Display(ctx, args[0])
						}
						return nil
					},
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
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							groupsService.DisplayEntity(ctx, args[0])
						}
						return nil
					},
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
					var outp interface{}
					if err := ctx.Request("GET", "health", nil, &outp); err != nil {
						ctx.Log.Err("Error on Check Health: %v\n", err)
					} else {
						ctx.Log.PP("Health info", outp)
					}
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
			ArgsUsage:   "<name> [password]",
			Description: "if password is not given as an argument, user will be prompted to enter it",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "client, c", Usage: "authenticate with oauth2 client ID and secret"},
			},
			Action: func(c *cli.Context) error {
				if a, ctx := initCmd(cfg, c, 1, 2, false, nil); ctx != nil {
					prompt, path, login := "Password", vidmLoginPath, LoginSystemUser
					if c.Bool("client") {
						prompt, path, login = "Secret", vidmTokenPath, ClientCredentialsGrant
					}
					pwd := getArgOrPassword(cfg.Log, prompt, a[1], false)
					if authHeader, err := login(ctx, path, a[0], pwd); err != nil {
						cfg.Log.Err("Error getting access token: %v\n", err)
					} else if cfg.WithAuthHeader(authHeader).Save() {
						cfg.Log.Info("Access token saved\n")
					}
				}
				return nil
			},
		},
		{
			Name: "logout", Usage: "deletes access token from configuration store for current target",
			Action: func(c *cli.Context) error {
				if args := initArgs(cfg, c, 0, 0, nil); args != nil && cfg.WithAuthHeader("").Save() {
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
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							rolesService.DisplayEntity(ctx, args[0])
						}
						return nil
					},
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
			Name: "target", Usage: "set or display the target workspace instance",
			ArgsUsage: "[newTargetURL] [targetName]",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force, f", Usage: "force target -- don't validate URL with health check"},
				cli.BoolFlag{Name: "delete, d", Usage: "delete specified or current target"},
				cli.BoolFlag{Name: "delete-all", Usage: "delete all targets"},
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
						cfg.SetTarget(args[0], args[1], nil)
					} else {
						cfg.SetTarget(args[0], args[1], checkTarget)
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
			Name: "tenant", Usage: "gets/sets tenant configuration", ArgsUsage: "<tenantName> [key=value]...",
			Action: func(c *cli.Context) error {
				if args, ctx := initCmd(cfg, c, 1, -1, true, nil); ctx != nil {
					CmdTenantConfig(ctx, args[0], args[1:])
				}
				return nil
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
							if err := usersService.AddEntity(ctx, user); err != nil {
								ctx.Log.Err("Error creating user '%s': %v\n", user.Name, err)
							} else {
								ctx.Log.Info(fmt.Sprintf("User '%s' successfully added\n", user.Name))
							}
						}
						return nil
					},
				},
				{
					Name: "get", Usage: "display user account", ArgsUsage: "<userName>",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							usersService.DisplayEntity(ctx, args[0])
						}
						return nil
					},
				},
				{
					Name: "delete", Usage: "delete user account", ArgsUsage: "<userName>",
					Action: func(c *cli.Context) error {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							usersService.DeleteEntity(ctx, args[0])
						}
						return nil
					},
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
		{
			Name: "schema", Usage: "get SCIM schema of specific type", ArgsUsage: "<type>",
			Description: "Supported types are User, Group, Role, PasswordState, ServiceProviderConfig\n",
			Action: func(c *cli.Context) error {
				if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
					CmdSchema(ctx, args[0])
				}
				return nil
			},
		},
	}

	if err = app.Run(args); err != nil {
		fmt.Fprintln(errorW, "failed to run app: ", err)
	}
}
