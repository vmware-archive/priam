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
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

const NoTarget = ""

// target is used to encapsulate everything needed to connect to a vidm instance.
type Target struct {
	Host       string
	AuthHeader string `yaml:",omitempty"`
}

type Config struct {
	CurrentTarget string
	Targets       map[string]Target
	fileName      string
	Log           *Logr `yaml:"-"`
}

func GetYamlFile(filename string, output interface{}) error {
	if f, err := ioutil.ReadFile(filename); err != nil {
		return err
	} else {
		return yaml.Unmarshal(f, output)
	}
}

func PutYamlFile(filename string, input interface{}) error {
	if f, err := yaml.Marshal(input); err != nil {
		return err
	} else {
		return ioutil.WriteFile(filename, f, 0644)
	}
}

func (cfg *Config) Init(log *Logr, fileName string) bool {
	if err := GetYamlFile(fileName, cfg); err != nil && !os.IsNotExist(err) {
		log.Err("could not read config file %s, error: %v\n", fileName, err)
		return false
	}

	// get yaml file clears all fields, so these must be set after unmarshalling
	cfg.Log, cfg.fileName = log, fileName

	if cfg.Targets == nil {
		cfg.Targets = make(map[string]Target)
		cfg.CurrentTarget = NoTarget
	} else if cfg.CurrentTarget == NoTarget || cfg.Targets[cfg.CurrentTarget] == (Target{}) {
		cfg.CurrentTarget = NoTarget
	}
	return true
}

func (cfg *Config) Save() bool {
	if err := PutYamlFile(cfg.fileName, cfg); err != nil {
		cfg.Log.Err("could not write config file %s, error: %v\n", cfg.fileName, err)
		return false
	}
	return true
}

func (cfg *Config) Clear() {
	cfg.CurrentTarget = NoTarget
	cfg.Targets = nil
	if cfg.Save() {
		cfg.Log.Info("all targets deleted.\n")
	}
}

func (cfg *Config) PrintTarget(prefix string) {
	if cfg.CurrentTarget == NoTarget {
		cfg.Log.Info("no target set\n")
	} else {
		cfg.Log.Info("%s target is: %s, %s\n", prefix, cfg.CurrentTarget,
			cfg.Targets[cfg.CurrentTarget].Host)
	}
}

func (cfg *Config) ClearTarget(name string) {
	if cfg.CurrentTarget == name {
		cfg.CurrentTarget = NoTarget
	}
	delete(cfg.Targets, name)
	if cfg.Save() {
		cfg.Log.Info("deleted target %s.\n", name)
	}
}

func (cfg *Config) hasTarget(name string) bool {
	_, ok := cfg.Targets[name]
	return ok
}

func (cfg *Config) WithAuthHeader(hdr string) *Config {
	tgt := cfg.Targets[cfg.CurrentTarget]
	tgt.AuthHeader = hdr
	cfg.Targets[cfg.CurrentTarget] = tgt
	return cfg
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
func (cfg *Config) findTarget(url, name string) string {
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
	return NoTarget
}

func (cfg *Config) DeleteTarget(url, name string) {
	if url == "" {
		if cfg.CurrentTarget == NoTarget {
			cfg.Log.Info("nothing deleted, no target set\n")
		} else {
			cfg.ClearTarget(cfg.CurrentTarget)
		}
	} else if tgt := cfg.findTarget(url, name); tgt == NoTarget {
		cfg.Log.Info("nothing deleted, no such target found\n")
	} else {
		cfg.ClearTarget(tgt)
	}
}

func (cfg *Config) SetTarget(url, name string, checkURL func(*Config) bool) {
	if url == "" {
		return
	}

	if tgt := cfg.findTarget(url, name); tgt != NoTarget {
		// found existing target
		cfg.CurrentTarget = tgt
		if cfg.Save() {
			cfg.PrintTarget("new")
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
	cfg.Targets[cfg.CurrentTarget] = Target{Host: ensureFullURL(url)}
	if (checkURL == nil || checkURL(cfg)) && cfg.Save() {
		cfg.PrintTarget("new")
	}
}

func (cfg *Config) ListTargets() {
	var keys []string
	for k := range cfg.Targets {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		cfg.Log.Info("name: %s\nhost: %s\n\n", k, cfg.Targets[k].Host)
	}
	cfg.PrintTarget("current")
}
