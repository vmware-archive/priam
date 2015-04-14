package main

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
)

type wksApp struct {
	Name           string `yaml:"name,omitempty"`
	Uuid           string `yaml:"uuid,omitempty"`
	PackageVersion string `yaml:"packageVersion,omitempty"`
	Description    string `yaml:"description,omitempty"`
	//IconFile              string                 `yaml:"iconFile,omitempty"`
	ResourceConfiguration map[string]interface{} `yaml:"resourceConfiguration,omitempty"`
	AccessPolicy          string                 `yaml:"accessPolicy,omitempty"`
	AccessPolicySetUuid   string                 `yaml:"accessPolicySet Uuid,omitempty"`
	CatalogItemType       string                 `yaml:"catalogItemType,omitempty"`
	Labels                []string               `yaml:"catalogItemType,omitempty"`
	AuthInfo              map[string]interface{} `yaml:"authInfo,omitempty"`
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

func GetAccessPolicyUuid(name string) string {
	return name
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
	for _, v := range manifest.Applications {
		var w = &v.Workspace
		if w.AccessPolicySetUuid == "" {
			if w.AccessPolicy == "" {
				log(lerr, "Invalid manifest entry for %s: one of accessPolicy name or AccessPolicySetUuid must be specified\n", w.Name)
				continue
			}
			w.AccessPolicySetUuid = GetAccessPolicyUuid(w.AccessPolicy)
		} else if w.AccessPolicy != "" {
			log(lerr, "Invalid manifest for %s: both accessPolicy \"%s\" and AccessPolicySetUuid \"%s\" cannot be specified\n",
				w.Name, w.AccessPolicy, w.AccessPolicySetUuid)
			continue
		}
		if w.Uuid == "" {
			w.Uuid = uuid.New()
		}

	}
	//log(linfo, "manifest is %#v\n", manifest)
	logpp(linfo, "manifest", manifest)
}

func cmdAppDel(c *cli.Context) {
	if args, authHdr := InitCmd(c, 1); authHdr != "" {
		path := fmt.Sprintf("jersey/manager/api/catalogitems/%s", args[0])
		if err := httpReq("DELETE", tgtURL(path), InitHdrs(authHdr), nil, nil); err != nil {
			log(lerr, "Error deleting app %s from catalog: %v\n", args[0], err)
		} else {
			log(linfo, "app %s deleted\n", args[0])
		}
	}
}

func cmdAppList(c *cli.Context) {
	summaryFields := []string{"Apps", "name", "description", "catalogItemType", "uuid"}
	count, filter := c.Int("count"), c.String("filter")
	if count == 0 {
		count = 1000
	}
	path := fmt.Sprintf("jersey/manager/api/catalogitems/search?pageSize=%v", count)
	input := struct {
		NameFilter string `json:"nameFilter,omitempty"`
	}{filter}
	if authHdr := authHeader(); authHdr != "" {
		body := make(map[string]interface{})
		hdrs := InitHdrs(authHdr, "catalog.summary.list", "catalog.search")
		if err := httpReq("POST", tgtURL(path), hdrs, &input, &body); err != nil {
			log(lerr, "Error: %v\n", err)
		} else {
			logppf(linfo, "Apps", body["items"], summaryFields)
		}
	}
}
