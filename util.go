package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
)

type logType int

const (
	linfo logType = iota
	lerr
	ldebug
	ltrace
)

type logStyle int

const (
	ljson logStyle = iota
	lyaml
)

var debugMode, traceMode, verboseMode bool
var logStyleDefault = lyaml

func log(lt logType, format string, args ...interface{}) {
	switch lt {
	case linfo:
		fmt.Printf(format, args...)
	case lerr:
		fmt.Fprintf(os.Stderr, format, args...)
	case ldebug:
		if debugMode {
			fmt.Printf(format, args...)
		}
	case ltrace:
		if traceMode {
			fmt.Printf(format, args...)
		}
	}
}

func toJson(input interface{}) (output []byte, err error) {
	switch inp := input.(type) {
	case nil:
	case string:
		output = []byte(inp)
	case []byte:
		output = inp
	case *string:
		if inp != nil {
			output = []byte(*inp)
		}
	case *[]byte:
		if inp != nil {
			output = *inp
		}
	default:
		output, err = json.Marshal(inp)
	}
	return
}

func toStringWithStyle(ls logStyle, input interface{}) string {
	var err error
	var outp interface{}
	var out, inp []byte
	var raw string
	if input != nil {
		switch in := input.(type) {
		case string:
			inp = []byte(in)
			raw = in
		case *string:
			inp = []byte(*in)
			raw = *in
		case []byte:
			inp = in
			raw = string(in)
		default:
			outp = input
		}
	}
	if outp == nil {
		if inp == nil || len(inp) == 0 {
			return ""
		}
		err = json.Unmarshal(inp, &outp)
	}
	if err == nil {
		if ls == lyaml {
			out, err = yaml.Marshal(outp)
		} else {
			out, err = json.MarshalIndent(outp, "", "  ")
		}
	}
	if err != nil {
		if raw == "" {
			raw = fmt.Sprintf("%v", input)
		}
		return fmt.Sprintf("Could not pretty print: %v\nraw:\n%v", err, raw)
	}
	return string(out)
}

func hasString(s string, a []string) bool {
	for _, v := range a {
		if v == s {
			return true
		}
	}
	return false
}

func logWithFilter(lt logType, indent, label, sep string, info interface{}, filter []string) {
	const indenter, arrayPrefix string = "  ", "- "
	if label != "" && label != arrayPrefix && !hasString(label, filter) {
		return
	}
	switch inf := info.(type) {
	case []interface{}:
		log(lt, "%s%s%s\n", indent, label, sep)
		for _, v := range inf {
			logWithFilter(lt, indent, arrayPrefix, "", v, filter)
		}
	case map[string]interface{}:
		if label != arrayPrefix {
			log(lt, "%s%s%s\n", indent, label, sep)
			label = indenter
		}
		for _, k := range filter {
			if v, ok := inf[k]; ok {
				logWithFilter(lt, indent+label, k, ":", v, filter)
				if label == arrayPrefix {
					label = indenter
				}
			}
		}
	default:
		log(lt, "%s%s%s %v\n", indent, label, sep, inf)
	}

}

func logppf(lt logType, title string, info interface{}, filter []string) {
	if verboseMode || len(filter) == 0 {
		logpp(lt, title, info)
	} else {
		logWithFilter(lt, "", title, ":", info, filter)
	}
}

func logWithStyle(lt logType, ls logStyle, prefix string, input interface{}) {
	if s := toStringWithStyle(ls, input); s == "" {
		log(lt, "%s is empty.\n", prefix)
	} else {
		log(lt, "%s\n%s\n", prefix, s)
	}
}

func logpp(lt logType, prefix string, input interface{}) {
	logWithStyle(lt, logStyleDefault, prefix, input)
}

func getYamlFile(filename string, output interface{}) error {
	if f, err := ioutil.ReadFile(filename); err != nil {
		return err
	} else {
		return yaml.Unmarshal(f, output)
	}
}

func putYamlFile(filename string, input interface{}) error {
	if f, err := yaml.Marshal(input); err != nil {
		return err
	} else {
		return ioutil.WriteFile(filename, f, 0644)
	}
}

func stringOrDefault(v, dfault string) string {
	if v != "" {
		return v
	}
	return dfault
}
