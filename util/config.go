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
	"io/ioutil"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

const NoTarget = ""
const HostOption = "host"
const InsecureSkipVerifyOption = "insecure_skip_verify"

/* Host modes definitions. */
const HostMode = "mode"
const (
	TenantInHost = "tenant-in-host"
	TenantInPath = "tenant-in-path"
)

/* Config represents a set of named targets, with an indication of which target is currently
   active. Each target contains a map of options. The only options known to this code are HostOption, HostMode, InsecureSkipVerify
   all other options are up to the users of the config struct.
*/
type Config struct {
	CurrentTarget string
	Targets       map[string]map[string]interface{}
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

/* YAML allows keys in maps to be datatypes other than string, whereas JSON only supports
   keys that are strings. The YAML parser defaults to keys of type interface{}. This
   function will take an object produced by the YAML parser and converts any map keys to
   strings so that the object can be rendered as JSON.
*/
func ChangeKeysToString(input interface{}) interface{} {
	switch inp := input.(type) {
	case []interface{}:
		output := make([]interface{}, len(inp))
		for i, v := range inp {
			output[i] = ChangeKeysToString(v)
		}
		return output
	case map[interface{}]interface{}:
		output := make(map[string]interface{})
		for k, v := range inp {
			output[fmt.Sprintf("%v", k)] = ChangeKeysToString(v)
		}
		return output
	case map[string]interface{}:
		output := make(map[string]interface{})
		for k, v := range inp {
			output[k] = ChangeKeysToString(v)
		}
		return output
	}
	return input
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
		cfg.Targets = make(map[string]map[string]interface{})
		cfg.CurrentTarget = NoTarget
	} else if cfg.CurrentTarget == NoTarget || cfg.Targets[cfg.CurrentTarget] == nil {
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
			cfg.Targets[cfg.CurrentTarget][HostOption])
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

func (cfg *Config) Option(name string) string {
	if value, ok := cfg.Targets[cfg.CurrentTarget][name].(string); ok {
		return value
	}
	return ""
}

func (cfg *Config) OptionAsBool(name string) bool {
	if value, ok := cfg.Targets[cfg.CurrentTarget][name].(bool); ok {
		return value
	}
	return false
}

func (cfg *Config) WithOptions(options map[string]interface{}) *Config {
	for k, v := range options {
		cfg.Targets[cfg.CurrentTarget][k] = v
	}
	return cfg
}

func (cfg *Config) WithoutOptions(optionKeys ...string) *Config {
	for _, k := range optionKeys {
		delete(cfg.Targets[cfg.CurrentTarget], k)
	}
	return cfg
}

func ensureFullURL(url string) string {
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		return url
	}
	return "https://" + url
}

// Returns true if the current vIDM targeted is set to use the host part of the URL to determine the tenant name
// If no host mode has been set (previous behaviour), then consider we are in "tenant in host" mode.
func (cfg *Config) IsTenantInHost() bool {
	return cfg.Option(HostMode) != TenantInPath || cfg.Option(HostMode) == ""
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
			if v[HostOption] == fullURL {
				return k
			}
		}
	} else {
		tgt, ok := cfg.Targets[name]
		if ok && tgt[HostOption] == fullURL {
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

func (cfg *Config) SetTarget(url, name string, insecureSkipVerify bool, checkURL func(*Config, *bool) bool) {
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

	// check tenant in path mode or not
	hostMode := TenantInHost
	if strings.Contains(url, "/SAAS/t/") {
		hostMode = TenantInPath
	}

	cfg.CurrentTarget = name
	cfg.Targets[cfg.CurrentTarget] = map[string]interface{}{HostOption: ensureFullURL(url), HostMode: hostMode}
	if insecureSkipVerify {
		cfg.Targets[cfg.CurrentTarget][InsecureSkipVerifyOption] = true
	}

	if (checkURL == nil || checkURL(cfg, &insecureSkipVerify)) && cfg.Save() {
		cfg.Log.Info("Mode detected: %s\n", hostMode)
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
		cfg.Log.Info("name: %s\nhost: %s\n\n", k, cfg.Targets[k][HostOption])
	}
	cfg.PrintTarget("current")
}
