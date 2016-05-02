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
package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"os"
)

type LogStyle int

const (
	LJson LogStyle = iota
	LYaml
)

type Logr struct {
	DebugOn, TraceOn, VerboseOn bool
	Style                       LogStyle
	ErrW, OutW                  io.Writer
}

func NewLogr() *Logr {
	return &Logr{false, false, false, LYaml, os.Stderr, os.Stdout}
}

func (l *Logr) ClearBuffers() *Logr {
	l.ErrW, l.OutW = &bytes.Buffer{}, &bytes.Buffer{}
	return l
}

func NewBufferedLogr() *Logr {
	return (&Logr{false, false, false, LYaml, nil, nil}).ClearBuffers()
}

func (l *Logr) InfoString() string {
	return l.OutW.(*bytes.Buffer).String()
}

func (l *Logr) ErrString() string {
	return l.ErrW.(*bytes.Buffer).String()
}

func (l *Logr) Info(format string, args ...interface{}) {
	fmt.Fprintf(l.OutW, format, args...)
}

func (l *Logr) Err(format string, args ...interface{}) {
	fmt.Fprintf(l.ErrW, format, args...)
}

func (l *Logr) Debug(format string, args ...interface{}) {
	if l.DebugOn {
		fmt.Fprintf(l.OutW, format, args...)
	}
}

func (l *Logr) Trace(format string, args ...interface{}) {
	if l.TraceOn {
		fmt.Fprintf(l.OutW, format, args...)
	}
}

func ToStringWithStyle(ls LogStyle, input interface{}) string {
	var err error
	var outp []byte
	if ls == LYaml {
		outp, err = yaml.Marshal(input)
	} else {
		outp, err = json.MarshalIndent(input, "", "  ")
	}
	if err != nil {
		return fmt.Sprintf("%v", input)
	}
	return string(outp)
}

// when JSON or YAML data are parsed into a general interface{}, they
// produce a nested object of interface{}, []interface{} and
// map[string]interface{} -- arrays and maps of various types of data.
// This method takes such an object and removes any map keys that are
// not in the filter. returns a new filtered interface{}
//
func (l *Logr) Filter(info interface{}, filter []string) interface{} {
	switch inf := info.(type) {
	case []interface{}:
		filteredArray := make([]interface{}, 0)
		for _, v := range inf {
			if fv := l.Filter(v, filter); fv != nil {
				filteredArray = append(filteredArray, fv)
			}
		}
		if len(filteredArray) == 0 {
			return nil
		}
		return filteredArray
	case map[string]interface{}:
		filteredMap := make(map[string]interface{}, len(inf))
		for _, k := range filter {
			if v, ok := inf[k]; ok {
				if fv := l.Filter(v, filter); fv != nil {
					filteredMap[k] = fv
				}
			}
		}
		if len(filteredMap) == 0 {
			return nil
		}
		return filteredMap
	}
	return info
}

// pp will pretty print in json or yaml format (based on logr.style) to logr.info.
// If filter is not empty and logr is not verbose, output will only include map
// values with those keys.
func (l *Logr) PP(title string, info interface{}, filter ...string) {
	if !l.VerboseOn && len(filter) > 0 {
		info = l.Filter(info, filter)
	}
	l.Info("---- %s ----\n%s", title, ToStringWithStyle(l.Style, info))
}
