package main

import (
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"fmt"
)

type wksApp struct {
	Name           string                 `yaml:"name,omitempty"`
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

func cmdAppAdd(c *cli.Context) {
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
	//log(linfo, "manifest is %#v\n", manifest)
	logpp(linfo, "manifest", manifest)
}

func cmdAppDel(c *cli.Context) {
}

func cmdAppList(c *cli.Context) {
	count, filter := c.Int("count"), c.String("filter")
	if count == 0 {
		count = 1000
	}
	path := fmt.Sprintf("jersey/manager/api/catalogitems/search?pageSize=%v", count)
	input := struct{NameFilter string `json:"nameFilter,omitempty"`} {filter}
	if authHdr := authHeader(); authHdr != "" {
		var body string
		hdrs := InitHdrs(authHdr, "catalog.summary.list", "catalog.search")
		if err := httpReq("POST", tgtURL(path), hdrs, &input, &body); err != nil {
			log(lerr, "Error: %v\n", err)
		} else {
			logpp(linfo, "Application catalog", body)
		}
	}
}
