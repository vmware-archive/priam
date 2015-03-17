package main

import (
	"fmt"
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

var inR io.Reader = os.Stdin
var outW io.Writer = os.Stdout
var errW io.Writer = os.Stderr
var debugMode, traceMode bool

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
