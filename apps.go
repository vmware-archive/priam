package main

import (
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/codegangsta/cli"
	"strings"
)

type wksApp struct {
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
	Workspace wksApp
}

type itemResponse struct {
	Links map[string]interface{}   `json:"_links,omitempty" yaml:"_links,omitempty"`
	Items []map[string]interface{} `json:",omitempty" yaml:",omitempty"`
}

func accessPolicyId(name, authHdr string) string {
	body := new(itemResponse)
	if err := httpReq("GET", tgtURL("accessPolicies"), InitHdrs(authHdr, "accesspolicyset.list"), nil, &body); err != nil {
		log(lerr, "Error getting access policies: %v\n", err)
		return ""
	}
	for _, item := range body.Items {
		if name == "" && item["base"] == true || caselessEqual(name, item["name"]) {
			if s, ok := item["uuid"].(string); ok {
				return s
			}
		}
	}
	log(lerr, "Could not find access policy uuid\n")
	return ""
}

// input name, uuid
// output uuid of existing app with the input uuid or uuid of first app with name
func checkAppExists(name, uuid, authHdr string) (outid string, err error) {
	path, body := "catalogitems/search?pageSize=10000", new(itemResponse)
	hdrs := InitHdrs(authHdr, "catalog.summary.list", "catalog.search")
	if err = httpReq("POST", tgtURL(path), hdrs, "{}", &body); err == nil {
		for _, item := range body.Items {
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

func getAppUuid(name, authHdr string) (uuid, mtype string, err error) {
	input := fmt.Sprintf(`{"nameFilter":"%s"}`, escapeQuotes(name))
	path, body := "catalogitems/search?pageSize=10000", new(itemResponse)
	hdrs := InitHdrs(authHdr, "catalog.summary.list", "catalog.search")
	if err = httpReq("POST", tgtURL(path), hdrs, &input, &body); err == nil {
		for _, item := range body.Items {
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

func getAppByUuid(uuid, mtype, authHdr string) (app map[string]interface{}, err error) {
	app = make(map[string]interface{})
	path := fmt.Sprintf("catalogitems/%s", uuid)
	err = httpReq("GET", tgtURL(path), InitHdrs(authHdr, mtype), nil, &app)
	return
}

func maybeEntitle(authHdr, itemID, subjName, subjType, nameAttr, appName string) {
	if subjName != "" {
		subjID, err := scimNameToID(strings.Title(subjType+"s"), nameAttr, subjName, authHdr)
		if err == nil {
			err = entitleSubject(authHdr, subjID, strings.ToUpper(subjType+"s"), itemID)
		}
		if err != nil {
			log(lerr, "Could not entitle %s \"%s\" to app \"%s\", error: %v\n", subjType, subjName, appName, err)
		} else {
			log(linfo, "Entitled %s \"%s\" to app \"%s\".\n", subjType, subjName, appName)
		}
	}
}

func publishApps(authHdr, manifile string) {
	if manifile == "" {
		manifile = "manifest.yaml"
	}
	var manifest struct{ Applications []manifestApp }
	if err := getYamlFile(manifile, &manifest); err != nil {
		log(lerr, "Error getting manifest: %v\n", err)
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
		if w.Name == "" {
			w.Name = v.Name
		}
		method, path, errVerb, successVerb := "POST", "catalogitems", "adding", "added"
		id, err := checkAppExists(w.Name, w.Uuid, authHdr)
		if err != nil {
			log(lerr, "Error checking if app %s exists: %v\n", w.Name, err)
			continue
		}
		if id != "" {
			method, errVerb, successVerb, w.Uuid = "PUT", "updating", "updated", id
			path += "/" + id
		}
		if w.Uuid == "" {
			w.Uuid = uuid.New()
		}
		amtype := "catalog." + strings.ToLower(w.CatalogItemType)
		cmtype, iconFile, entitleGrp, entitleUser := amtype, w.IconFile, w.EntitleGroup, w.EntitleUser
		w.IconFile, w.EntitleGroup, w.EntitleUser = "", "", ""
		content, err := toJson(w)
		if err != nil {
			log(lerr, "Error converting app %s to JSON: %v\n", w.Name, err)
			continue
		}
		if iconFile != "" {
			if content, cmtype, err = newReqWithFileUpload("catalogitem", amtype, content, iconFile); err != nil {
				log(lerr, "Error creating upload request for app %s: %v\n", w.Name, err)
				continue
			}
		}
		hdrs := InitHdrs(authHdr, amtype, cmtype)
		if err = httpReq(method, tgtURL(path), hdrs, content, nil); err != nil {
			log(lerr, "Error %s %s to the catalog: %v\n", errVerb, w.Name, err)
		} else {
			log(linfo, "App \"%s\" %s to the catalog\n", w.Name, successVerb)
		}
		maybeEntitle(authHdr, w.Uuid, entitleGrp, "group", "displayName", w.Name)
		maybeEntitle(authHdr, w.Uuid, entitleUser, "user", "userName", w.Name)
	}
}

func cmdAppAdd(c *cli.Context) {
	if args, authHdr := InitCmd(c, 0, 1); authHdr != "" {
		publishApps(authHdr, args[0])
	}
}

func cmdAppDel(c *cli.Context) {
	if args, authHdr := InitCmd(c, 1, 1); authHdr != "" {
		if uuid, _, err := getAppUuid(args[0], authHdr); err != nil {
			log(lerr, "Error getting app info by name: %v\n", err)
		} else if err := httpReq("DELETE", tgtURL(fmt.Sprintf("catalogitems/%s", uuid)), InitHdrs(authHdr), nil, nil); err != nil {
			log(lerr, "Error deleting app %s from catalog: %v\n", args[0], err)
		} else {
			log(linfo, "app %s deleted\n", args[0])
		}
	}
}

func cmdAppGet(c *cli.Context) {
	if args, authHdr := InitCmd(c, 1, 1); authHdr != "" {
		if uuid, mtype, err := getAppUuid(args[0], authHdr); err != nil {
			log(lerr, "Error getting app info by name: %v\n", err)
		} else if app, err := getAppByUuid(uuid, mtype, authHdr); err != nil {
			log(lerr, "Error getting app info by uuid: %v\n", err)
		} else {
			logpp(linfo, "App "+args[0], app)
		}
	}
}

func cmdAppList(c *cli.Context) {
	summaryFields := []string{"Apps", "name", "description", "catalogItemType", "uuid"}
	count, filter := c.Int("count"), c.String("filter")
	if count == 0 {
		count = 1000
	}
	path, input := fmt.Sprintf("catalogitems/search?pageSize=%v", count), "{}"
	if filter != "" {
		input = fmt.Sprintf(`{"nameFilter":"%s"}`, escapeQuotes(filter))
	}
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
