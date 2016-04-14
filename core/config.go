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
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

const noTarget = ""

// target is used to encapsulate everything needed to connect to a vidm instance.
type target struct {
	Host                   string
	ClientID, ClientSecret string `yaml:",omitempty"`
}

type config struct {
	CurrentTarget string
	Targets       map[string]target
	fileName      string
	log           *logr
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

func newAppConfig(log *logr, fileName string) *config {
	appCfg := &config{}
	if err := getYamlFile(fileName, appCfg); err != nil && !os.IsNotExist(err) {
		log.err("could not read config file %s, error: %v\n", fileName, err)
		return nil
	}

	// get yaml file clears all fields, so these must be set after unmarshalling
	appCfg.log, appCfg.fileName = log, fileName

	if appCfg.Targets == nil {
		appCfg.Targets = make(map[string]target)
		appCfg.CurrentTarget = noTarget
	} else if appCfg.CurrentTarget == noTarget || appCfg.Targets[appCfg.CurrentTarget] == (target{}) {
		appCfg.CurrentTarget = noTarget
	}
	return appCfg
}

func (cfg *config) save() bool {
	if err := putYamlFile(cfg.fileName, cfg); err != nil {
		cfg.log.err("could not write config file %s, error: %v\n", cfg.fileName, err)
		return false
	}
	return true
}

func (cfg *config) clear() {
	cfg.CurrentTarget = noTarget
	cfg.Targets = nil
	if cfg.save() {
		cfg.log.info("all targets deleted.\n")
	}
}

func (cfg *config) printTarget(prefix string) {
	if cfg.CurrentTarget == noTarget {
		cfg.log.info("no target set\n")
	} else {
		cfg.log.info("%s target is: %s, %s\n", prefix, cfg.CurrentTarget,
			cfg.Targets[cfg.CurrentTarget].Host)
	}
}

func (cfg *config) clearTarget(name string) {
	if cfg.CurrentTarget == name {
		cfg.CurrentTarget = noTarget
	}
	delete(cfg.Targets, name)
	if cfg.save() {
		cfg.log.info("deleted target %s.\n", name)
	}
}

func (cfg *config) hasTarget(name string) bool {
	_, ok := cfg.Targets[name]
	return ok
}

func ensureFullURL(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return "https://" + url
}

// findTarget attempts to find an existing target based on user input. User
// may specify a target as a url followed by an optional name, or with no input
// to specify the current target. findTarget returns the name of any existing
// target.
func (cfg *config) findTarget(url, name string) string {
	fullURL := ensureFullURL(url)
	if name == "" {
		// if url is already a key and no name is given, assume url is a name
		if cfg.hasTarget(url) {
			return url
		}

		// no name given, look up first target that matches url
		for k, v := range cfg.Targets {
			if v.Host == fullURL {
				return k
			}
		}
	} else {
		tgt, ok := cfg.Targets[name]
		if ok && tgt.Host == fullURL {
			return name
		}

	}
	return noTarget
}

func (cfg *config) deleteTarget(url, name string) {
	if url == "" {
		if cfg.CurrentTarget == noTarget {
			cfg.log.info("nothing deleted, no target set\n")
		} else {
			cfg.clearTarget(cfg.CurrentTarget)
		}
	} else if tgt := cfg.findTarget(url, name); tgt == noTarget {
		cfg.log.info("nothing deleted, no such target found\n")
	} else {
		cfg.clearTarget(tgt)
	}
}

func (cfg *config) setTarget(url, name string, checkURL func(*config) bool) {
	if url == "" {
		return
	}

	if tgt := cfg.findTarget(url, name); tgt != noTarget {
		// found existing target
		cfg.CurrentTarget = tgt
		if cfg.save() {
			cfg.printTarget("new")
		}
		return
	}

	// if no name given, make one up.
	if name == "" {
		for i := 0; ; i++ {
			k := fmt.Sprintf("%v", i)
			if !cfg.hasTarget(k) {
				name = k
				break
			}
		}
	}

	cfg.CurrentTarget = name
	cfg.Targets[cfg.CurrentTarget] = target{Host: ensureFullURL(url)}
	if (checkURL == nil || checkURL(cfg)) && cfg.save() {
		cfg.printTarget("new")
	}
}

func (cfg *config) listTargets() {
	var keys []string
	for k := range cfg.Targets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cfg.log.info("name: %s\nhost: %s\n\n", k, cfg.Targets[k].Host)
	}
	cfg.printTarget("current")
}
