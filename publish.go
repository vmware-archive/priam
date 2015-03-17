package main

import (
	"github.com/codegangsta/cli"
)

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

func cmdPublish(c *cli.Context) {
	println("push app command")
	//getManifest()
}
