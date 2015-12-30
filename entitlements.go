package main

import (
	"fmt"
)

const fmtEntitlement = `
{
  "returnPayloadOnError" : true,
  "operations" : [ {
    "method" : "POST",
    "data" : {
      "catalogItemId" : "%s",
      "subjectType" : "%s",
      "subjectId" : "%s",
      "activationPolicy" : "AUTOMATIC"
    }
  } ]
}`

func entitleSubject(ctx *httpContext, subjectId, subjectType, itemID string) error {
	inp := fmt.Sprintf(fmtEntitlement, itemID, subjectType, subjectId)
	ctx.accept("bulk.sync.response").contentType("entitlements.definition.bulk")
	return ctx.request("POST", "entitlements/definitions", inp, nil)
}

func getEntitlement(ctx *httpContext, rtypeName, name string) {
	var resType, id string
	body := make(map[string]interface{})
	switch rtypeName {
	case "user":
		resType, id = "users", scimNameToID(ctx, "Users", "userName", name)
	case "group":
		resType, id = "groups", scimNameToID(ctx, "Groups", "displayName", name)
	case "app":
		resType, id = "catalogitems", name
	default:
		ctx.log.err("First parameter must be user, group or app\n")
		return
	}
	if id == "" {
		return
	}
	path := fmt.Sprintf("entitlements/definitions/%s/%s", resType, id)
	if err := ctx.request("GET", path, nil, &body); err != nil {
		ctx.log.err("Error: %v\n", err)
	} else {
		ctx.log.ppf("Entitlements", body["items"], []string{"Entitlements",
			"catalogItemId", "subjectType", "subjectId", "activationPolicy"})
	}
}
