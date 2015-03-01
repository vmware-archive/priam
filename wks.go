package main

import (
	//"golang.org/x/oauth2"
	"fmt"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

type target struct {
	Name         string
	ClientID     string
	ClientSecret string
	Host         string
	Current      bool
}

var appConfig struct {
	DebugMode bool
	Targets   []target
}

var currentTarget uint
var appConfigFile string = "./wks.yaml"

type wksAppConfig struct {
	packageVersion string
	description    string
	iconFile       string
	attributeMaps  map[string]string
	accessPolicy   string
	authInfo       map[string]string
}

type manifest struct {
	name      string
	memory    string
	instances int
	path      string
	buildpack string
	env       map[string]string
	workspace wksAppConfig
}

func getAppConfig() {
	getYAML("./wks.yaml", &appConfig)
	/*
		var cfg map[interface{}]interface{}
		getYAML("./wks.yaml", &cfg)
		for _, av := range cfg["targets"].([]interface{}) {
			t := av.(map[interface{}]interface{})
			//fmt.Printf("t %#v\n", t)
			tgt := target{ystr(t["name"]), ystr(t["clientid"]),
				ystr(t["clientsecret"]), ystr(t["host"]),
				ybool(t["current"])}
			//fmt.Printf("target %#v\n", tgt)
			targets = append(targets, tgt)
		}
		//fmt.Printf("targets %#v\n", targets)
	*/
}

func putAppConfig() {
	putYAML("./wks_out.yaml", appConfig)
}

func getManifest() {
	var cfg map[interface{}]interface{}
	getYAML("./manifest.yaml", &cfg)
	fmt.Printf("\n\nAppValue: %#v\n\n", cfg["applications"])
	for _, app := range cfg["applications"].([]interface{}) {
		for k, v := range app.(map[interface{}]interface{}) {
			fmt.Printf("key %v, Value: %#v\n\n", k, v)
		}
	}
}

func cmdPush(c *cli.Context) {
	println("push app command")
	getManifest()
}

func cmdTarget(c *cli.Context) {
	//fmt.Printf()
	println("target command")
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

func wks(args []string, inCfg []bytes, outCfg []bytes, out ioutil.Writer, error ioutl.Writer) (error uint) {

}

func main() {
	app := cli.NewApp()
	app.Name = "wks"
	app.Usage = "general usage goes here"
	app.Action = cli.ShowAppHelp
	app.Email = ""
	app.Author = ""

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
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "force, f",
					Value: "",
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

	getAppConfig()
	putAppConfig()
	url, err := url.Parse("what.com/")
	fmt.Printf("url: %#v, err: %v", url, err)

	app.Run(os.Args)
}


