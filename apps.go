package main

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"strings"
)

type priamApp struct {
	Name                  string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Uuid                  string                 `json:"uuid,omitempty" yaml:"uuid,omitempty"`
	PackageVersion        string                 `json:"packageVersion,omitempty" yaml:"packageVersion,omitempty"`
	Description           string                 `json:"description,omitempty" yaml:"description,omitempty"`
	IconFile              string                 `json:"iconFile,omitempty" yaml:"iconFile,omitempty"`
	EntitleGroup          string                 `json:"entitleGroup,omitempty" yaml:"entitleGroup,omitempty"`
	EntitleUser           string                 `json:"entitleUser,omitempty" yaml:"entitleUser,omitempty"`
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
	Workspace priamApp
}

type itemResponse struct {
	Links map[string]interface{}   `json:"_links,omitempty" yaml:"_links,omitempty"`
	Items []map[string]interface{} `json:",omitempty" yaml:",omitempty"`
}

func accessPolicyId(ctx *httpContext, name string) string {
	outp := &itemResponse{}
	ctx.accept("accesspolicyset.list")
	if err := ctx.request("GET", "accessPolicies", nil, &outp); err != nil {
		ctx.log.err("Error getting access policies: %v\n", err)
		return ""
	}
	for _, item := range outp.Items {
		if name == "" && item["base"] == true || caselessEqual(name, item["name"]) {
			if s, ok := item["uuid"].(string); ok {
				return s
			}
		}
	}
	ctx.log.err("Could not find access policy uuid\n")
	return ""
}

// input name, uuid
// output uuid of existing app with the input uuid or uuid of first app with name
func checkAppExists(ctx *httpContext, name, uuid string) (outid string, err error) {
	outp := &itemResponse{}
	ctx.accept("catalog.summary.list").contentType("catalog.search")
	if err = ctx.request("POST", "catalogitems/search?pageSize=10000", "{}", &outp); err == nil {
		for _, item := range outp.Items {
			if caseEqual(uuid, item["uuid"]) {
				outid = uuid
				return
			}
			if caselessEqual(name, item["name"]) && outid == "" {
				outid = interfaceToString(item["uuid"])
			}
		}
	}
	return
}

func getAppUuid(ctx *httpContext, name string) (uuid, mtype string, err error) {
	inp, outp := fmt.Sprintf(`{"nameFilter":"%s"}`, escapeQuotes(name)), new(itemResponse)
	ctx.accept("catalog.summary.list").contentType("catalog.search")
	if err = ctx.request("POST", "catalogitems/search?pageSize=10000", &inp, &outp); err == nil {
		for _, item := range outp.Items {
			if u, ok := item["uuid"].(string); ok && caselessEqual(name, item["name"]) {
				if uuid != "" {
					err = fmt.Errorf("Multiple apps with name \"%s\"", name)
					return
				}
				uuid = u
				if mt, ok := item["catalogItemType"].(string); ok {
					mtype = "catalog." + strings.ToLower(mt)
				}
			}
		}
	}
	if uuid == "" {
		err = fmt.Errorf("No app found with name \"%s\"", name)
	}
	return
}

func getAppByUuid(ctx *httpContext, uuid, mtype string) (app map[string]interface{}, err error) {
	app = make(map[string]interface{})
	err = ctx.accept(mtype).request("GET", fmt.Sprintf("catalogitems/%s", uuid), nil, &app)
	return
}

func publishApps(ctx *httpContext, manifile string) {
	if manifile == "" {
		manifile = "manifest.yaml"
	}
	var manifest struct{ Applications []manifestApp }
	if err := getYamlFile(manifile, &manifest); err != nil {
		ctx.log.err("Error getting manifest: %v\n", err)
		return
	}
	for _, v := range manifest.Applications {
		var w = &v.Workspace
		if w.AccessPolicySetUuid == "" {
			if w.AccessPolicySetUuid = accessPolicyId(ctx, w.AccessPolicy); w.AccessPolicySetUuid == "" {
				continue
			}
			w.AccessPolicy = ""
		} else if w.AccessPolicy != "" {
			ctx.log.err("Invalid manifest for %s: both accessPolicy \"%s\" and AccessPolicySetUuid \"%s\" cannot be specified\n",
				w.Name, w.AccessPolicy, w.AccessPolicySetUuid)
			continue
		}
		if w.Name == "" {
			w.Name = v.Name
		}
		method, path, errVerb, successVerb := "POST", "catalogitems", "adding", "added"
		id, err := checkAppExists(ctx, w.Name, w.Uuid)
		if err != nil {
			ctx.log.err("Error checking if app %s exists: %v\n", w.Name, err)
			continue
		}
		if id != "" {
			method, errVerb, successVerb, w.Uuid = "PUT", "updating", "updated", id
			path += "/" + id
		}
		if w.Uuid == "" {
			w.Uuid = uuid.New()
		}
		mtype := "catalog." + strings.ToLower(w.CatalogItemType)
		iconFile, entitleGrp, entitleUser := w.IconFile, w.EntitleGroup, w.EntitleUser
		w.IconFile, w.EntitleGroup, w.EntitleUser = "", "", ""
		content, err := toJson(w)
		if err != nil {
			ctx.log.err("Error converting app %s to JSON: %v\n", w.Name, err)
			continue
		}
		if iconFile == "" {
			err = ctx.contentType(mtype).request(method, path, content, nil)
		} else {
			err = ctx.fileUploadRequest(method, path, "catalogitem", mtype, content, iconFile, nil)
		}
		if err != nil {
			ctx.log.err("Error %s %s to the catalog: %v\n", errVerb, w.Name, err)
			continue
		}
		ctx.log.info("App \"%s\" %s to the catalog\n", w.Name, successVerb)
		maybeEntitle(ctx, w.Uuid, entitleGrp, "group", "displayName", w.Name)
		maybeEntitle(ctx, w.Uuid, entitleUser, "user", "userName", w.Name)
	}
}

func appDelete(ctx *httpContext, name string) {
	if uuid, _, err := getAppUuid(ctx, name); err != nil {
		ctx.log.err("Error getting app info by name: %v\n", err)
	} else if err := ctx.request("DELETE", fmt.Sprintf("catalogitems/%s", uuid), nil, nil); err != nil {
		ctx.log.err("Error deleting app %s from catalog: %v\n", name, err)
	} else {
		ctx.log.info("app %s deleted\n", name)
	}
}

func appGet(ctx *httpContext, name string) {
	if uuid, mtype, err := getAppUuid(ctx, name); err != nil {
		ctx.log.err("Error getting app info by name: %v\n", err)
	} else if app, err := getAppByUuid(ctx, uuid, mtype); err != nil {
		ctx.log.err("Error getting app info by uuid: %v\n", err)
	} else {
		ctx.log.pp("App "+name, app)
	}
}

func appList(ctx *httpContext, count int, filter string) {
	summaryFields := []string{"Apps", "name", "description", "catalogItemType", "uuid"}
	if count == 0 {
		count = 1000
	}
	path, input := fmt.Sprintf("catalogitems/search?pageSize=%v", count), "{}"
	if filter != "" {
		input = fmt.Sprintf(`{"nameFilter":"%s"}`, escapeQuotes(filter))
	}
	body := make(map[string]interface{})
	ctx.accept("catalog.summary.list").contentType("catalog.search")
	if err := ctx.request("POST", path, input, &body); err != nil {
		ctx.log.err("Error: %v\n", err)
	} else {
		ctx.log.ppf("Apps", body["items"], summaryFields)
	}
}
