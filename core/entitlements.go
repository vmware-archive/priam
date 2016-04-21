/*
Copyright (c) 2016 VMware, Inc. All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package core

import (
	"fmt"
	. "priam/util"
	"strings"
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

// Create entitlement for the given user or group
func maybeEntitle(ctx *HttpContext, itemID, subjName, subjType, nameAttr, appName string) {
	if subjName != "" {
		subjID, err := scimGetID(ctx, strings.Title(subjType+"s"), nameAttr, subjName)
		if err == nil {
			err = entitleSubject(ctx, subjID, strings.ToUpper(subjType+"s"), itemID)
		}
		if err != nil {
			ctx.Log.Err("Could not entitle %s \"%s\" to app \"%s\", error: %v\n", subjType, subjName, appName, err)
		} else {
			ctx.Log.Info("Entitled %s \"%s\" to app \"%s\".\n", subjType, subjName, appName)
		}
	}
}

func entitleSubject(ctx *HttpContext, subjectId, subjectType, itemID string) error {
	inp := fmt.Sprintf(fmtEntitlement, itemID, subjectType, subjectId)
	ctx.Accept("bulk.sync.response").ContentType("entitlements.definition.bulk")
	return ctx.Request("POST", "entitlements/definitions", inp, nil)
}

// Get entitlement for the given user whose username is 'name'
// rtypeName has been validated before and is one of 'user', 'group' or 'app'
func GetEntitlement(ctx *HttpContext, rtypeName, name string) {
	var resType, id string
	body := make(map[string]interface{})
	switch rtypeName {
	case "user":
		resType, id = "users", scimNameToID(ctx, "Users", "userName", name)
	case "group":
		resType, id = "groups", scimNameToID(ctx, "Groups", "displayName", name)
	case "app":
		resType, id = "catalogitems", name
	}
	if id == "" {
		return
	}
	path := fmt.Sprintf("entitlements/definitions/%s/%s", resType, id)
	if err := ctx.Request("GET", path, nil, &body); err != nil {
		ctx.Log.Err("Error: %v\n", err)
	} else {
		ctx.Log.PPF("Entitlements", body["items"], "Entitlements",
			"catalogItemId", "subjectType", "subjectId", "activationPolicy")
	}
}
