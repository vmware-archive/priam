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
	"github.com/stretchr/testify/assert"
	. "github.com/vmware/priam/testaid"
	"testing"
)

var coffeeService = &OauthResourceService{"Coffee", "/espressomachine", "coffee", "coffee.list",
	[]string{"items", "sugar", "size", "name"}}

const coffeeItem = `{"name": "%s", "size": "single", "sugar": true, "alias": "espresso", "gift": ""}`
const coffeeList = `{"items": [` + coffeeItem + `]}`

func TestCoffeeGet(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + coffeeService.path + "/for_elsa": func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, coffeeService.itemMT+"+json", req.Accept)
			return &TstReply{Status: 200, Output: fmt.Sprintf(coffeeItem, "for_elsa"), ContentType: coffeeService.itemMT + "+json"}
		}})
	defer srv.Close()
	coffeeService.Get(ctx, "for_elsa")
	AssertOnlyInfoContains(t, ctx, "name: for_elsa")
	AssertOnlyInfoContains(t, ctx, "sugar: true")
}

func TestCoffeeNotFound(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + coffeeService.path + "/for_hans": func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, coffeeService.itemMT+"+json", req.Accept)
			return &TstReply{Status: 404, Output: `{"message":"coffee.not.found"}`,
				ContentType: "application/json"}
		}})
	defer srv.Close()
	coffeeService.Get(ctx, "for_hans")
	AssertOnlyErrorContains(t, ctx, `Error: 404 Not Found`)
	AssertOnlyErrorContains(t, ctx, `coffee.not.found`)
}

func TestCoffeeList(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + coffeeService.path: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, coffeeService.listMT+"+json", req.Accept)
			return &TstReply{Status: 200, Output: fmt.Sprintf(coffeeList, "for_sven"), ContentType: coffeeService.listMT + "+json"}
		}})
	defer srv.Close()
	coffeeService.List(ctx)
	AssertOnlyInfoContains(t, ctx, `items`)
	AssertOnlyInfoContains(t, ctx, `name: for_sven`)
	AssertOnlyInfoContains(t, ctx, `size: single`)
}

func TestCoffeeListRejected(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"GET" + coffeeService.path: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, coffeeService.listMT+"+json", req.Accept)
			return &TstReply{Status: 403}
		}})
	defer srv.Close()
	coffeeService.List(ctx)
	AssertOnlyErrorContains(t, ctx, `Error: 403 Forbidden`)
}

func TestCoffeeDelete(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"DELETE" + coffeeService.path + "/for_hans": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Status: 200}
		}})
	defer srv.Close()
	coffeeService.Delete(ctx, "for_hans")
	AssertOnlyInfoContains(t, ctx, `Coffee "for_hans" deleted`)
}

func TestCoffeeDeleteRejected(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"DELETE" + coffeeService.path + "/for_elsa": func(t *testing.T, req *TstReq) *TstReply {
			return &TstReply{Status: 500, Output: "elsa's feeling mean"}
		}})
	defer srv.Close()
	coffeeService.Delete(ctx, "for_elsa")
	AssertOnlyErrorContains(t, ctx, `Error deleting Coffee "for_elsa": 500 Internal Server Error`)
	AssertOnlyErrorContains(t, ctx, `feeling mean`)
}

func TestCoffeeAdd(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"POST" + coffeeService.path: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, coffeeService.itemMT+"+json", req.ContentType)
			assert.Equal(t, `{"name":"for_anna","size":"double","sugar":false}`, req.Input)
			return &TstReply{Status: 200, Output: coffeeItem, ContentType: coffeeService.itemMT}
		}})
	defer srv.Close()
	coffeeService.Add(ctx, "for_anna", map[string]interface{}{"name": "for_anna", "sugar": false, "size": "double"})
	AssertOnlyInfoContains(t, ctx, `Successfully added Coffee "for_anna"`)
}

func TestCoffeeRejected(t *testing.T) {
	srv, ctx := NewTestContext(t, map[string]TstHandler{
		"POST" + coffeeService.path: func(t *testing.T, req *TstReq) *TstReply {
			assert.Equal(t, coffeeService.itemMT+"+json", req.ContentType)
			return &TstReply{Status: 406, ContentType: "application/json",
				Output: `{"message":"elsa's insecure and rejects sven's clumsy attempt at kindness"}`}
		}})
	defer srv.Close()
	coffeeService.Add(ctx, "for_elsa", map[string]interface{}{"name": "for_elsa",
		"size": "single", "sugar": true, "gift": "from_sven"})
	AssertOnlyErrorContains(t, ctx, `clumsy`)
	AssertOnlyErrorContains(t, ctx, `406`)
}
