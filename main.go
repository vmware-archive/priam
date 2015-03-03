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
	DebugMode bool
	CurrentTarget string
	Targets   map[string]target
}

var appCfg appConfig
var inR io.Reader = os.Stdin
var outW io.Writer = os.Stdout
var errW io.Writer = os.Stderr
var inCfg, outCfg, manifest []byte 

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
	return 
}

func putAppConfig() (err error) {
	outCfg, err = yaml.Marshal(&appCfg)
	return
}

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

func cmdPush(c *cli.Context) {
	println("push app command")
	getManifest()
}

func cmdTarget(c *cli.Context) {
	a := c.Args()
	if len(a) < 1 {
		if appCfg.CurrentTarget == "" {
			fmt.Fprintf(outW, "no target set\n")

		} else {
			fmt.Fprintf(outW, "Current target is: %s\n", 
				appCfg.Targets[appCfg.CurrentTarget].Host)
		}
		return
	}
	if !strings.HasPrefix(a[0], "http:") && !strings.HasPrefix(a[0], "https:") {
		a[0] = "https://" + a[0]
	}
	if !c.Bool("force") {
		//health check here
	}
	appCfg.CurrentTarget = "2"
	appCfg.Targets[appCfg.CurrentTarget] = target{Host: a[0]}
	putAppConfig()
	fmt.Fprintf(outW, "New target is: %s\n", 
		appCfg.Targets[appCfg.CurrentTarget].Host)
}

func cmdHealth(c *cli.Context) {
	println("health:\n\n")
	resp, err := http.Get("https://radio.workspaceair.com/SAAS/jersey/manager/api/health")
	if err != nil {
		fmt.Printf("Error on GET: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error on ReadAll: %v\n", err)
		return
	}
	fmt.Printf("Body: %v\n", string(body))
}

func wks(args []string) (err error) {
	app := cli.NewApp()
	app.Name = "wks"
	app.Usage = "a utility to publish applications to Workspace"
	app.Action = cli.ShowAppHelp
	app.Email = ""
	app.Author = ""
	app.Writer = outW

	app.Commands = []cli.Command{
		{
			Name:      "health",
			ShortName: "h",
			Usage:     "check workspace service health",
			Action:    cmdHealth,
		},
		{
			Name:      "push",
			ShortName: "p",
			Usage:     "push an application",
			Action:    cmdPush,
		},
		{
			Name:      "target",
			ShortName: "t",
			Usage:     "set or display the target workspace instance",
			Action:    cmdTarget,
			Description: "wks target [new target URL] [targetName]",
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "force, f",
					Usage: "force target even if workspace instance not reachable",
				},
			},
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


