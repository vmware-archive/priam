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
	"strings"
)

func stringOrDefault(v, dfault string) string {
	if v != "" {
		return v
	}
	return dfault
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func interfaceToString(i interface{}) string {
	if s, ok := i.(string); ok {
		return s
	}
	return ""
}

func caselessEqual(s string, i interface{}) bool {
	if t, ok := i.(string); ok {
		return strings.EqualFold(s, t)
	}
	return false
}

func caseEqual(s string, i interface{}) bool {
	if t, ok := i.(string); ok {
		return s == t
	}
	return false
}

// Returns true if the given string is one of the strings in the given array, false otherwise
func hasString(s string, a []string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}
