package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"path/filepath"
)

func getYAML(filename string, out interface{}) {
	fullname, _ := filepath.Abs(filename)
	yamlFile, err := ioutil.ReadFile(fullname)
	if err != nil {
		panic(err)
	}
	if err = yaml.Unmarshal(yamlFile, out); err != nil {
		panic(err)
	}
	return
}

func putYAML(filename string, in interface{}) {
	fullname, _ := filepath.Abs(filename)
	y, err := yaml.Marshal(in)
	if err != nil {
		panic(err)
	}
	if err := ioutil.WriteFile(fullname, y, 0644); err != nil {
		panic(err)
	}
	return
}

func ystr(in interface{}) (out string) {
	out, err := in.(string)
	if err == false {
		out = ""
	}
	return
}

func ybool(in interface{}) (out bool) {
	out, err := in.(bool)
	if err == false {
		out = false
	}
	return
}
