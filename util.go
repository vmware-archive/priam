package main

import (
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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

var inR io.Reader = os.Stdin
var outW io.Writer = os.Stdout
var errW io.Writer = os.Stderr
var debugMode, traceMode bool
var logStyleDefault = lyaml

func log(lt logType, format string, args ...interface{}) {
	switch lt {
	case linfo:
		fmt.Fprintf(outW, format, args...)
	case lerr:
		fmt.Fprintf(errW, format, args...)
	case ldebug:
		if debugMode {
			fmt.Fprintf(outW, format, args...)
		}
	case ltrace:
		if traceMode {
			fmt.Fprintf(outW, format, args...)
		}
	}
}

func logWithStyle(lt logType, ls logStyle, prefix string, input interface{}) {
	var err error
	var outp interface{}
	var out, inp []byte
	if input != nil {
		switch in := input.(type) {
		case string:
			inp = []byte(in)
		case *string:
			inp = []byte(*in)
		case []byte:
			inp = in
		default:
			outp = input
		}
	}
	if outp == nil {
		if inp == nil || len(inp) == 0 {
			log(lt, "%s is empty.\n", prefix)
			return
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
	if err == nil {
		log(lt, "%s\n%s\n", prefix, string(out))
	} else {
		log(lt, "%s:\nCould not pretty print: %v\nraw:\n%v", prefix, err, input)
	}
}

func logpp(lt logType, prefix string, input interface{}) {
	logWithStyle(lt, logStyleDefault, prefix, input)
}

func getFile(dir, filename string) (out []byte, err error) {
	fullname, err := filepath.Abs(filepath.Join(dir, filename))
	if err == nil {
		out, err = ioutil.ReadFile(fullname)
	} else if strings.HasSuffix(err.Error(), "no such file or directory") {
		out = []byte{}
	}
	return
}

func putFile(dir, filename string, in []byte) (err error) {
	fullname, err := filepath.Abs(filepath.Join(dir, filename))
	if err == nil {
		err = ioutil.WriteFile(fullname, in, 0644)
	}
	return
}
