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
