package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
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

func (l *logr) filter(indent, label, sep string, info interface{}, filter []string) {
	const indenter, arrayPrefix string = "  ", "- "
	if label != "" && label != arrayPrefix && !hasString(label, filter) {
		return
	}
	switch inf := info.(type) {
	case []interface{}:
		l.info("%s%s%s\n", indent, label, sep)
		for _, v := range inf {
			l.filter(indent, arrayPrefix, "", v, filter)
		}
	case map[string]interface{}:
		if label != arrayPrefix {
			l.info("%s%s%s\n", indent, label, sep)
			label = indenter
		}
		for _, k := range filter {
			if v, ok := inf[k]; ok {
				l.filter(indent+label, k, ":", v, filter)
				if label == arrayPrefix {
					label = indenter
				}
			}
		}
	default:
		l.info("%s%s%s %v\n", indent, label, sep, inf)
	}

}

func (l *logr) pp(prefix string, input interface{}) {
	if s := toStringWithStyle(l.style, input); s == "" {
		l.info("%s is empty.\n", prefix)
	} else {
		l.info("%s\n%s\n", prefix, s)
	}
}

func (l *logr) ppf(title string, info interface{}, filter []string) {
	if l.verboseOn || len(filter) == 0 {
		l.pp(title, info)
	} else {
		l.filter("", title, ":", info, filter)
	}
}
