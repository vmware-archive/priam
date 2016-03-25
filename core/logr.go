package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
	"strings"
)

type logStyle int

const (
	ljson logStyle = iota
	lyaml
)

type logr struct {
	debugOn, traceOn, verboseOn bool
	style                       logStyle
	errw, outw                  io.Writer
}

func newLogr() *logr {
	return &logr{false, false, false, lyaml, os.Stderr, os.Stdout}
}

func newBufferedLogr() *logr {
	return &logr{false, false, false, lyaml, &bytes.Buffer{}, &bytes.Buffer{}}
}

func (l *logr) infoString() string {
	return l.outw.(*bytes.Buffer).String()
}

func (l *logr) errString() string {
	return l.errw.(*bytes.Buffer).String()
}

func (l *logr) info(format string, args ...interface{}) {
	fmt.Fprintf(l.outw, format, args...)
}

func (l *logr) err(format string, args ...interface{}) {
	fmt.Fprintf(l.errw, format, args...)
}

func (l *logr) debug(format string, args ...interface{}) {
	if l.debugOn {
		fmt.Fprintf(l.outw, format, args...)
	}
}

func (l *logr) trace(format string, args ...interface{}) {
	if l.traceOn {
		fmt.Fprintf(l.outw, format, args...)
	}
}

func toStringWithStyle(ls logStyle, input interface{}) string {
	var err error
	var outp []byte
	if ls == lyaml {
		outp, err = yaml.Marshal(input)
	} else {
		outp, err = json.MarshalIndent(input, "", "  ")
	}
	if err != nil {
		return fmt.Sprintf("%v", input)
	}
	return string(outp)
}

// helper function for filter()
func parseIndent(p string) int {
	lines := strings.Split(p, "\n")
	lastLine := lines[len(lines)-1]
	if strings.HasSuffix(lastLine, "- ") {
		return len(lastLine) - 2
	}
	return strings.IndexFunc(lastLine, func(r rune) bool { return r != ' ' })
}

// when JSON or YAML data are parsed into a general interface{}, they
// produce a nested object of interface{}, []interface{} and
// map[string]interface{} -- arrays and maps of various types of data.
// This method takes such an object and pretty-prints it somewhat like
// YAML, but it filters and orders the output.
//
// prefix is one of more lines to be printed before a selected object
// info is the object of parsed JSON or YAML
// filter is an array of strings. Only map elements with keys in the
// filter will be printed. Sibling map elements will be printed in the
// order of the keys in the filter. To print a nested key, the keys of
// the parent elements must be included.
//
// returns true if something was printed
//
func (l *logr) filter(prefix string, info interface{}, filter []string) (printed bool) {
	nextPrefix := ""
	printValue := func(key, sep string, value interface{}) {
		if printed {
			nextPrefix = fmt.Sprintf("%*s%s%s", parseIndent(nextPrefix), "", key, sep)
		} else if strings.HasSuffix(prefix, ": ") {
			nextPrefix = fmt.Sprintf("%s\n%*s%s%s", prefix, parseIndent(prefix)+2, "", key, sep)
		} else {
			nextPrefix = fmt.Sprintf("%s%s%s", prefix, key, sep)
		}
		if l.filter(nextPrefix, value, filter) {
			printed = true
		}
	}
	switch inf := info.(type) {
	case []interface{}:
		for _, v := range inf {
			printValue("", "- ", v)
		}
	case map[string]interface{}:
		for _, k := range filter {
			if v, ok := inf[k]; ok {
				printValue(k, ": ", v)
			}
		}
	default:
		l.info("%s%v\n", prefix, inf)
		printed = true
	}
	return
}

func (l *logr) pp(prefix string, input interface{}) {
	l.info("---- %s ----\n%s", prefix, toStringWithStyle(l.style, input))
}

func (l *logr) ppf(title string, info interface{}, filter []string) {
	if l.verboseOn || len(filter) == 0 {
		l.pp(title, info)
	} else {
		l.info("---- %s ----\n", title)
		l.filter("", info, filter)
	}
}
