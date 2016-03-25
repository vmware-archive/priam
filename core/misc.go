package core

import (
	"fmt"
	"net/url"
	"strings"
)

func cmdLocalUserStore(ctx *HttpContext, args []string) {
	const desc = "Local User Store configuration"
	const path = "localuserstore"
	const mtype = "local.userstore"
	if len(args) == 0 {
		ctx.getPrintJson(desc, path, mtype)
		return
	}
	keyvals, outp := make(map[string]interface{}), ""
	for _, arg := range args {
		kv := strings.SplitAfterN(arg, "=", 2)
		keyvals[strings.TrimSuffix(kv[0], "=")] = kv[1]
	}
	ctx.accept(mtype).contentType(mtype)
	if err := ctx.request("PUT", path, keyvals, &outp); err != nil {
		ctx.log.err("Error: %v\n", err)
	} else {
		ctx.log.pp(desc, outp)
	}
}

func cmdTenantConfig(ctx *HttpContext, name string, nvpairs []string) {
	const desc = "Tenant configuration"
	const mtype = "tenants.tenant.config.list"
	path := fmt.Sprintf("tenants/tenant/%s/config", name)
	type nvpair struct {
		Name  string            `json:"name"`
		Value string            `json:"value"`
		Links map[string]string `json:"_links"`
	}
	if len(nvpairs) == 0 {
		ctx.getPrintJson(desc, path, mtype)
		return
	}
	keyvals, outp := []nvpair{}, ""
	for _, arg := range nvpairs {
		kv := strings.SplitAfterN(arg, "=", 2)
		keyvals = append(keyvals, nvpair{strings.TrimSuffix(kv[0], "="), kv[1], map[string]string{}})
	}
	ctx.accept(mtype).contentType(mtype)
	if err := ctx.request("PUT", path, keyvals, &outp); err != nil {
		ctx.log.err("Error: %v\n", err)
	} else {
		ctx.log.pp(desc, outp)
	}
}

func cmdSchema(ctx *HttpContext, name string) {
	vals := make(url.Values)
	vals.Set("filter", fmt.Sprintf("name eq \"%s\"", name))
	path := fmt.Sprintf("scim/Schemas?%v", vals.Encode())
	ctx.getPrintJson("Schema for "+name, path, "")
}
