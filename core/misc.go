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
	. "github.com/vmware/priam/util"
	"net/url"
	"strings"
)

func CmdLocalUserStore(ctx *HttpContext, args []string) {
	const desc = "Local User Store configuration"
	const path = "localuserstore"
	const mtype = "local.userstore"
	if len(args) == 0 {
		ctx.GetPrintJson(desc, path, mtype)
		return
	}
	keyvals, outp := make(map[string]interface{}), ""
	for _, arg := range args {
		kv := strings.SplitAfterN(arg, "=", 2)
		keyvals[strings.TrimSuffix(kv[0], "=")] = kv[1]
	}
	ctx.Accept(mtype).ContentType(mtype)
	if err := ctx.Request("PUT", path, keyvals, &outp); err != nil {
		ctx.Log.Err("Error: %v\n", err)
	} else {
		ctx.Log.PP(desc, outp, "name", "showLocalUserStore", "associatedIdPNames", "syncClient",
			"userStoreNameUsedForAuth", "uuid")
	}
}

func CmdTenantConfig(ctx *HttpContext, name string, nvpairs []string) {
	const desc = "Tenant configuration"
	const mtype = "tenants.tenant.config.list"
	path := fmt.Sprintf("tenants/tenant/%s/config", name)
	type nvpair struct {
		Name  string            `json:"name"`
		Value string            `json:"value"`
		Links map[string]string `json:"_links"`
	}
	if len(nvpairs) == 0 {
		ctx.GetPrintJson(desc, path, mtype)
		return
	}
	keyvals, outp := []nvpair{}, ""
	for _, arg := range nvpairs {
		kv := strings.SplitAfterN(arg, "=", 2)
		keyvals = append(keyvals, nvpair{strings.TrimSuffix(kv[0], "="), kv[1], map[string]string{}})
	}
	ctx.Accept(mtype).ContentType(mtype)
	if err := ctx.Request("PUT", path, keyvals, &outp); err != nil {
		ctx.Log.Err("Error: %v\n", err)
	} else {
		ctx.Log.PP(desc, outp)
	}
}

func CmdSchema(ctx *HttpContext, name string) {
	vals := make(url.Values)
	vals.Set("filter", fmt.Sprintf("name eq \"%s\"", name))
	path := fmt.Sprintf("scim/Schemas?%v", vals.Encode())
	ctx.GetPrintJson("Schema for "+name, path, "")
}
