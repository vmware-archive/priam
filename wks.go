package main

import (
	//"golang.org/x/oauth2"
	"fmt"
	"github.com/codegangsta/cli"
	yaml "github.com/smallfish/simpleyaml"
	"io/ioutil"
	"net/http"
	"os"
    "path/filepath"
)

func boom(c *cli.Context) {
	println("command was run with no params... print help here?")
}

func health(c *cli.Context) {
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


func main() {
	app := cli.NewApp()
	app.Name = "wks"
	app.Usage = "general usage goes here"
	app.Action = boom

	filename, _ := filepath.Abs("./manifest.yaml")
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	y, err := yaml.NewYaml(yamlFile)
	if err != nil {
		panic(err)
	}

	applications, err := y.Get("applications").Array()
	if err != nil {
		panic(err)
	}

	for i, _ := range applications {
		fmt.Printf("index: %v, value: %#v\n", i, y.Get("applications").GetIndex(i).Get("name"))
	}

	//println(y.Get("applications").GetIndex(0).Get("name").String())
	//config := make(map[string]interface{})

	//err = yaml.Unmarshal(yamlFile, &config)
	//if err != nil {
	//	panic(err)
	//}

	//fmt.Printf("Value: %#v\n\n", config)
	//fmt.Printf("Value: %#v\n", config["applications"])

	//appdata := make([]map[string]interface{})
	//apps := config["applications"].([]interface{})
	//app0 := apps[0].(map[interface{}]interface{})
	//app0s := app0.(map[string]interface{})
	//println(app0["name"].(string))
	//println(app0["name"])

	//fmt.Printf("Value: %#v\n", config["applications"])
    //for k := range config {
    //    println("key: ", k)
    //}

	app.Commands = []cli.Command{
		{
			Name:      "health",
			ShortName: "h",
			Usage:     "check workspace service health",
			Action:    health,
		},
		{
			Name:      "complete",
			ShortName: "c",
			Usage:     "complete a task on the list",
			Action: func(c *cli.Context) {
				println("completed task: ", c.Args().First())
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

	app.Run(os.Args)
}
