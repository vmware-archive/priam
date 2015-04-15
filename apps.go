package main

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/codegangsta/cli"
	"gopkg.in/yaml.v2"
	"strings"
)

type wksApp struct {
	Name           string `json:"name,omitempty" yaml:"name,omitempty"`
	Uuid           string `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	PackageVersion string `json:"packageVersion,omitempty" yaml:"packageVersion,omitempty"`
	Description    string `json:"description,omitempty" yaml:"description,omitempty"`
	//IconFile              string                 `json:"iconFile,omitempty" yaml:"iconFile,omitempty"`
	ResourceConfiguration map[string]interface{} `json:"resourceConfiguration" yaml:"resourceConfiguration,omitempty"`
	AccessPolicy          string                 `json:"accessPolicy,omitempty" yaml:"accessPolicy,omitempty"`
	AccessPolicySetUuid   string                 `json:"accessPolicySetUuid,omitempty" yaml:"accessPolicySetUuid,omitempty"`
	CatalogItemType       string                 `json:"catalogItemType,omitempty" yaml:"catalogItemType,omitempty"`
	Labels                []string               `json:"labels,omitempty" yaml:"labels,omitempty"`
	AuthInfo              map[string]interface{} `json:"authInfo,omitempty" yaml:"authInfo,omitempty"`
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

func accessPolicyId(name, authHdr string) string {
	body := make(map[string]interface{})
	if err := httpReq("GET", tgtURL("jersey/manager/api/accessPolicies"), InitHdrs(authHdr), nil, &body); err != nil {
		log(lerr, "Error getting access policies: %v\n", err)
		return ""
	}
	if items, ok := body["items"].([]interface{}); ok {
		for _, v := range items {
			if item, ok := v.(map[string]interface{}); ok {
				if name == "" && item["base"] == true || name == item["name"] {
					if s, ok := item["uuid"].(string); ok {
						return s
					}
				}
			}
		}
	}
	log(lerr, "Could not find access policy uuid\n")
	return ""
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
	authHdr := authHeader()
	if authHdr == "" {
		return
	}
	for _, v := range manifest.Applications {
		var w = &v.Workspace
		if w.AccessPolicySetUuid == "" {
			if w.AccessPolicySetUuid = accessPolicyId(w.AccessPolicy, authHdr); w.AccessPolicySetUuid == "" {
				continue
			}
			w.AccessPolicy = ""
		} else if w.AccessPolicy != "" {
			log(lerr, "Invalid manifest for %s: both accessPolicy \"%s\" and AccessPolicySetUuid \"%s\" cannot be specified\n",
				w.Name, w.AccessPolicy, w.AccessPolicySetUuid)
			continue
		}
		if w.Uuid == "" {
			w.Uuid = uuid.New()
		}
		if w.Name == "" {
			w.Name = v.Name
		}
		mtype := "catalog." + strings.ToLower(w.CatalogItemType)
		hdrs := InitHdrs(authHdr, mtype, mtype)
		if err := httpReq("POST", tgtURL("jersey/manager/api/catalogitems"), hdrs, w, nil); err != nil {
			log(lerr, "Error adding %s to the catalog: %v\n", w.Name, err)
		} else {
			log(linfo, "Apps %s added to the catalog\n", w.Name)
		}
	}
	//log(linfo, "manifest is %#v\n", manifest)
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
