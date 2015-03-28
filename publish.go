package main

import (
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
)

type wksApp struct {
	PackageVersion string                 `yaml:"packageVersion,omitempty"`
	Description    string                 `yaml:"description,omitempty"`
	IconFile       string                 `yaml:"iconFile,omitempty"`
	AttributeMaps  map[string]interface{} `yaml:"attributeMaps,omitempty"`
	AccessPolicy   string                 `yaml:"accessPolicy,omitempty"`
	AuthInfo       map[string]interface{} `yaml:"authInfo,omitempty"`
}

type manifestApp struct {
	Name      string
	Memory    string
	Instances int
	Path      string
	BuildPack string
	Env       map[string]string
	Workspace wksApp
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
	var manifest struct{ Applications []manifestApp }
	if yml, err := getFile(".", "manifest.yaml"); err == nil {
		if err := yaml.Unmarshal(yml, &manifest); err != nil {
			log(lerr, "Error parsing manifest: %v\n", err)
			return
		}
	} else {
		log(lerr, "Error opening manifest: %v\n", err)
		return
	}
	log(linfo, "manifest is %#v\n", manifest)
	ppJson(linfo, "manifest", manifest)
}
