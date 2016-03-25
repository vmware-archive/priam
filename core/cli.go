package core

import (
	"code.google.com/p/gopass"
	"fmt"
	"github.com/codegangsta/cli"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	vidmTokenPath = "/SAAS/API/1.0/oauth2/token"
	vidmBasePath = "/SAAS/jersey/manager/api/"
	vidmBaseMediaType = "application/vnd.vmware.horizon.manager."
)

// Directory service
var usersService DirectoryService = &SCIMUsersService{}
var groupsService DirectoryService = &SCIMGroupsService{}

func getPwd(prompt string) string {
	if s, err := gopass.GetPass(prompt); err != nil {
		panic(err)
	} else {
		return s
	}
}

func getArgOrPassword(log *logr, prompt, arg string, repeat bool) string {
	if arg != "" {
		return arg
	}
	for {
		if pwd := getPwd(prompt + ": "); !repeat || pwd == getPwd(prompt + " again: ") {
			return pwd
		}
		log.info(prompt + "s didn't match. Try again.")
	}
}

func initCtx(cfg *config, authn bool) *HttpContext {
	if cfg.CurrentTarget == "" {
		cfg.log.err("Error: no target set\n")
		return nil
	}
	tgt := cfg.Targets[cfg.CurrentTarget]
	ctx := newHttpContext(cfg.log, tgt.Host, vidmBasePath, vidmBaseMediaType)
	if authn {
		if err := ctx.clientCredsGrant(vidmTokenPath, tgt.ClientID, tgt.ClientSecret); err != nil {
			cfg.log.err("Error getting access token: %v\n", err)
			return nil
		}
	}
	return ctx
}

func initArgs(cfg *config, c *cli.Context, minArgs, maxArgs int, validateArgs func([]string) bool) []string {
	args := c.Args()
	if len(args) < minArgs {
		cfg.log.err("\nInput Error: at least %d arguments must be given\n\n", minArgs)
	} else if maxArgs >= 0 && len(args) > maxArgs {
		cfg.log.err("\nInput Error: at most %d arguments can be given\n\n", maxArgs)
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

func initCmd(cfg *config, c *cli.Context, minArgs, maxArgs int, authn bool, validateArgs func([]string) bool) (args []string, ctx *HttpContext) {
	args = initArgs(cfg, c, minArgs, maxArgs, validateArgs)
	if args != nil {
		ctx = initCtx(cfg, authn)
	}
	return
}

func initUserCmd(cfg *config, c *cli.Context, getPwd bool) (*basicUser, *HttpContext) {
	maxArgs := 1
	if getPwd {
		maxArgs = 2
	}
	args := initArgs(cfg, c, 1, maxArgs, nil)
	if args == nil {
		return nil, nil
	}
	user := &basicUser{Name: args[0], Given: c.String("given"),
		Family: c.String("family"), Email: c.String("email")}
	if getPwd {
		user.Pwd = getArgOrPassword(cfg.log, "Password", args[1], true)
	}
	return user, initCtx(cfg, true)
}

func checkTarget(cfg *config) bool {
	ctx, output := initCtx(cfg, false), ""
	if ctx == nil {
		return false
	}
	if err := ctx.request("GET", "health", nil, &output); err != nil {
		ctx.log.err("Error checking health of %s: \n", ctx.hostURL)
		return false
	}
	ctx.log.debug("health check output:\n%s\n", output)
	if !strings.Contains(output, "allOk") {
		ctx.log.err("Reply from %s does not meet health check\n", ctx.hostURL)
		return false
	}
	return true
}

func Priam(args []string, infoR io.Reader, infoW, errorW io.Writer) {
	var err error
	var cfg *config
	appName := filepath.Base(args[0])
	cli.HelpFlag.Usage = "show help for given command or subcommand"
	app, defaultCfgFile := cli.NewApp(), fmt.Sprintf(".%s.yaml", appName)
	app.Name, app.Usage = appName, "a utility to interact with VMware Identity Manager"
	app.Email, app.Author, app.Writer = "", "", infoW
	app.Action = cli.ShowAppHelp
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name: "config", Usage: "specify configuration file. Default: ~/" + defaultCfgFile,
		},
		cli.BoolFlag{Name: "debug, d", Usage: "print debug output"},
		cli.BoolFlag{Name: "json, j", Usage: "prefer output in json rather than yaml"},
		cli.BoolFlag{Name: "trace, t", Usage: "print all requests and responses"},
		cli.BoolFlag{Name: "verbose, V", Usage: "print verbose output"},
	}
	app.Before = func(c *cli.Context) (err error) {
		log := &logr{debugOn: c.Bool("debug"), traceOn: c.Bool("trace"),
			style: lyaml, verboseOn: c.Bool("verbose"), errw: errorW, outw: infoW}
		if c.Bool("json") {
			log.style = ljson
		}
		fileName := c.String("config")
		if fileName == "" {
			fileName = filepath.Join(os.Getenv("HOME"), defaultCfgFile)
		}
		if cfg = newAppConfig(log, fileName); cfg == nil {
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
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 0, 1, true, nil); ctx != nil {
							publishApps(ctx, args[0])
						}
					},
				},
				{
					Name: "delete", Usage: "delete an app from the catalog", ArgsUsage: "<appName>",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							appDelete(ctx, args[0])
						}
					},
				},
				{
					Name: "get", Usage: "get information about an app", ArgsUsage: "<appName>",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							appGet(ctx, args[0])
						}
					},
				},
				{
					Name: "list", Usage: "list all applications in the catalog", ArgsUsage: " ",
					Flags: pageFlags,
					Action: func(c *cli.Context) {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							appList(ctx, c.Int("count"), c.String("filter"))
						}
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
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 2, 2, true, func(args []string) bool {
							res := hasString(args[0], []string{"group", "user", "app"})
							if !res {
								cfg.log.err("First parameter of 'get' must be user, group or app\n")
							}
							return res
						}); ctx != nil {
							getEntitlement(ctx, args[0], args[1])
						}
					},
				},
			},
		},
		{
			Name: "group", Usage: "commands for groups",
			Subcommands: []cli.Command{
				{
					Name: "get", Usage: "get a specific group", ArgsUsage: "get <groupName>",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							groupsService.DisplayEntity(ctx, args[0])
						}
					},
				},
				{
					Name: "list", Usage: "list all groups", ArgsUsage: " ", Flags: pageFlags,
					Action: func(c *cli.Context) {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							groupsService.ListEntities(ctx, c.Int("count"), c.String("filter"))
						}
					},
				},
				{
					Name: "member", Usage: "add or remove users from a group",
					ArgsUsage: "<groupname> <username>", Flags: memberFlags,
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 2, 2, true, nil); ctx != nil {
							// @todo Put this in the interface
							scimMember(ctx, "Groups", "displayName", args[0], args[1], c.Bool("delete"))
						}
					},
				},
			},
		},
		{
			Name: "health", Usage: "check workspace service health", ArgsUsage: " ",
			Action: func(c *cli.Context) {
				if _, ctx := initCmd(cfg, c, 0, 0, false, nil); ctx != nil {
					var outp interface{}
					if err := ctx.request("GET", "health", nil, &outp); err != nil {
						ctx.log.err("Error on Check Health: %v\n", err)
					} else {
						ctx.log.pp("Health info", outp)
					}
				}
			},
		},
		{
			Name: "localuserstore", Usage: "gets/sets local user store configuration",
			ArgsUsage: "[key=value]...",
			Action: func(c *cli.Context) {
				if args, ctx := initCmd(cfg, c, 0, -1, true, nil); ctx != nil {
					cmdLocalUserStore(ctx, args)
				}
			},
		},
		{
			Name: "login", Usage: "validates and saves clientID and clientSecret",
			ArgsUsage:   "<clientID> [clientSecret]",
			Description: "if clientSecret is not given as an argument, user will be prompted to enter it",
			Action: func(c *cli.Context) {
				if a, ctx := initCmd(cfg, c, 1, 2, false, nil); ctx != nil {
					cfg.Targets[cfg.CurrentTarget] = target{Host: ctx.hostURL,
						ClientID: a[0], ClientSecret: getArgOrPassword(cfg.log, "Secret", a[1], false)}
					if ctx = initCtx(cfg, true); ctx != nil && cfg.save() {
						cfg.log.info("clientID and clientSecret saved\n")
					}
				}
			},
		},
		{
			Name: "policies", Usage: "get access policies", ArgsUsage: " ",
			Action: func(c *cli.Context) {
				if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
					ctx.getPrintJson("Access Policies", "accessPolicies", "accesspolicyset.list")
				}
			},
		},
		{
			Name: "role", Usage: "commands for roles",
			Subcommands: []cli.Command{
				{
					Name: "get", Usage: "get specific SCIM role", ArgsUsage: "<roleName>",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							scimGet(ctx, "Roles", "displayName", args[0])
						}
					},
				},
				{
					Name: "list", ArgsUsage: " ", Usage: "list all roles", Flags: pageFlags,
					Action: func(c *cli.Context) {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							scimList(ctx, c.Int("count"), c.String("filter"), "Roles")
						}
					},
				},
				{
					Name: "member", Usage: "add or remove users from a role",
					ArgsUsage: "<rolename> <username>", Flags: memberFlags,
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 2, 2, true, nil); ctx != nil {
							scimMember(ctx, "Roles", "displayName", args[0], args[1], c.Bool("delete"))
						}
					},
				},
			},
		},
		{
			Name: "target", Usage: "set or display the target workspace instance",
			ArgsUsage: "[newTargetURL] [targetName]",
			Flags: []cli.Flag{
				cli.BoolFlag{Name: "force, f", Usage: "force target -- don't validate URL with health check"},
			},
			Action: func(c *cli.Context) {
				if args := initArgs(cfg, c, 0, 2, nil); args != nil {
					if c.Bool("force") {
						cfg.target(args[0], args[1], nil)
					} else {
						cfg.target(args[0], args[1], checkTarget)
					}
				}
			},
		},
		{
			Name: "targets", Usage: "display all targets", ArgsUsage: " ",
			Action: func(c *cli.Context) {
				if initArgs(cfg, c, 0, 0, nil) != nil {
					cfg.targets()
				}
			},
		},
		{
			Name: "tenant", Usage: "gets/sets tenant configuration", ArgsUsage: "[key=value]...",
			Action: func(c *cli.Context) {
				if args, ctx := initCmd(cfg, c, 1, -1, true, nil); ctx != nil {
					cmdTenantConfig(ctx, args[0], args[1:])
				}
			},
		},
		{
			Name: "user", Usage: "user account commands",
			Subcommands: []cli.Command{
				{
					Name: "add", Usage: "create a user account", ArgsUsage: "<userName> [password]",
					Flags: userAttrFlags,
					Action: func(c *cli.Context) {
						if user, ctx := initUserCmd(cfg, c, true); ctx != nil {
							if err := usersService.AddEntity(ctx, user); err != nil {
								ctx.log.err("Error creating user '%s': %v\n", user.Name, err)
							} else {
								ctx.log.info(fmt.Sprintf("User '%s' successfully added\n", user.Name))
							}
						}
					},
				},
				{
					Name: "get", Usage: "display user account", ArgsUsage: "<userName>",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							usersService.DisplayEntity(ctx, args[0])
						}
					},
				},
				{
					Name: "delete", Usage: "delete user account", ArgsUsage: "<userName>",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							usersService.DeleteEntity(ctx, args[0])
						}
					},
				},
				{
					Name: "list", Usage: "list user accounts", ArgsUsage: " ",
					Flags: pageFlags,
					Action: func(c *cli.Context) {
						if _, ctx := initCmd(cfg, c, 0, 0, true, nil); ctx != nil {
							usersService.ListEntities(ctx, c.Int("count"), c.String("filter"))
						}
					},
				},
				{
					Name: "load", ArgsUsage: "<fileName>", Usage: "loads yaml file of an array of users.",
					Description: "Example yaml file content:\n---\n- {name: joe, given: joseph, pwd: changeme}\n" +
					"- {name: sue, given: susan, family: jones, email: sue@what.com}\n",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
							usersService.LoadEntities(ctx, args[0])
						}
					},
				},
				{
					Name: "password", Usage: "set a user's password", ArgsUsage: "<username> [password]",
					Description: "If password is not given as an argument, user will be prompted to enter it",
					Action: func(c *cli.Context) {
						if args, ctx := initCmd(cfg, c, 1, 2, true, nil); ctx != nil {
							usersService.UpdateEntity(ctx, args[0], &basicUser{Pwd: getArgOrPassword(cfg.log, "Password", args[1], true)})
						}
					},
				},
				{
					Name: "update", Usage: "update user account", ArgsUsage: "<userName>",
					Flags: userAttrFlags,
					Action: func(c *cli.Context) {
						if user, ctx := initUserCmd(cfg, c, false); ctx != nil {
							usersService.UpdateEntity(ctx, user.Name, user)
						}
					},
				},
			},
		},
		{
			Name: "schema", Usage: "get SCIM schema of specific type", ArgsUsage: "<type>",
			Description: "Supported types are User, Group, Role, PasswordState, ServiceProviderConfig\n",
			Action: func(c *cli.Context) {
				if args, ctx := initCmd(cfg, c, 1, 1, true, nil); ctx != nil {
					cmdSchema(ctx, args[0])
				}
			},
		},
	}

	if err = app.Run(args); err != nil {
		fmt.Fprintln(errorW, "failed to run app: ", err)
	}
}
