package main

import (
	//"golang.org/x/oauth2"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net/http"
	//"net/url"
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
var debugMode bool 

func putDebug(format string, args ...interface{}) {
	if debugMode {
		fmt.Fprintf(outW, format, args)
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

func checkHealth(host string) (body []byte, err error) {
	resp, err := http.Get(host + "/SAAS/jersey/manager/api/health")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	return
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

func cmdTarget(c *cli.Context) {
	a := c.Args()
	if len(a) < 1 {
		if appCfg.CurrentTarget == "" {
			fmt.Fprintf(outW, "no target set\n")

		} else {
			fmt.Fprintf(outW, "current target is: %s\nname: %s\n",
				appCfg.Targets[appCfg.CurrentTarget].Host, appCfg.CurrentTarget)
		}
		return
	}

	// if a[0] is a key, use it
	if appCfg.Targets[a[0]].Host != "" {
		appCfg.CurrentTarget = a[0]
	} else {

		if !strings.HasPrefix(a[0], "http:") && !strings.HasPrefix(a[0], "https:") {
			a[0] = "https://" + a[0]
		}

		// if an existing target uses this host a[0], set it
		reuseTarget := ""
		if len(a) < 2 {
			for k, v := range appCfg.Targets {
				if v.Host == a[0] {
					reuseTarget = k
					break
				}
			}
		}

		if reuseTarget != "" {
			appCfg.CurrentTarget = reuseTarget
		} else {

			if !c.Bool("force") {
				body, err := checkHealth(a[0])
				if err != nil {
					fmt.Fprintf(errW, "Error checking health of %s: \n", a[0])
					return
				}
				putDebug("health output: %s\n", string(body))
				if !strings.Contains(string(body), "allOk") {
					fmt.Println(string(body))
					fmt.Fprintf(errW, "Reply from %s does not look like Workspace\n", a[0])
					return
				}
			}
			if len(a) > 1 {
				appCfg.CurrentTarget = a[1]
			} else {
				// didn't specify a target name, make one up.
				for i := 0; ; i++ {
					k := fmt.Sprintf("%v", i)
					if appCfg.Targets[k].Host == "" {
						appCfg.CurrentTarget = k
						break
					}
				}
			}
			appCfg.Targets[appCfg.CurrentTarget] = target{Host: a[0]}
		}
	}
	putAppConfig()
	fmt.Fprintf(outW, "new target is: %s\nname: %s\n",
		appCfg.Targets[appCfg.CurrentTarget].Host, appCfg.CurrentTarget)
}

func cmdHealth(c *cli.Context) {
	body, err := checkHealth(appCfg.Targets[appCfg.CurrentTarget].Host)
	if err != nil {
		fmt.Fprintf(errW, "Error on Check Health: %v\n", err)
		return
	}
	fmt.Fprintf(outW, "Body: %v\n", string(body))
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
		return
	}

	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: "print debug out",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:      "health",
			ShortName: "h",
			Usage:     "check workspace service health",
			Action:    cmdHealth,
		},
		{
			Name:        "target",
			ShortName:   "t",
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
