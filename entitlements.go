package main

import (
	"fmt"
	"github.com/codegangsta/cli"
)

func cmdEntitlementGet(c *cli.Context) {
	var resType, id string
	body := make(map[string]interface{})
	args, authHdr := InitCmd(c, 2, 2)
	if authHdr == "" {
		return
	}
	switch args[0] {
	case "user":
		resType, id = "users", cmdNameToID("Users", "userName", args[1], authHdr)
	case "group":
		resType, id = "groups", cmdNameToID("Groups", "displayName", args[1], authHdr)
	case "app":
		resType, id = "catalogitems", args[1]
	default:
		log(lerr, "First parameter must be user, group or app\n")
		return
	}
	if id == "" {
		return
	}
	path := fmt.Sprintf("entitlements/definitions/%s/%s", resType, id)
	if err := httpReq("GET", tgtURL(path), InitHdrs(authHdr), nil, &body); err != nil {
		log(lerr, "Error: %v\n", err)
	} else {
		logppf(linfo, "Entitlements", body["items"], []string{"Entitlements",
			"catalogItemId", "subjectType", "subjectId", "activationPolicy"})
	}
}
