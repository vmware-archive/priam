package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"strings"
)

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
	appCfg := &config{log: log, fileName: fileName}
	if err := getYamlFile(fileName, appCfg); err != nil {
		if !os.IsNotExist(err) {
			log.err("could not read config file %s, error: %v\n", fileName, err)
			return nil
		}
	}
	if appCfg.CurrentTarget != "" &&
		appCfg.Targets[appCfg.CurrentTarget] != (target{}) {
		return appCfg
	}
	for k := range appCfg.Targets {
		appCfg.CurrentTarget = k
		return appCfg
	}
	if appCfg.Targets == nil {
		appCfg.Targets = make(map[string]target)
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

func (cfg *config) printTarget(prefix string) {
	if cfg.CurrentTarget == "" {
		cfg.log.info("no target set\n")
	} else {
		cfg.log.info("%s target is: %s, %s\n", prefix, cfg.CurrentTarget,
			cfg.Targets[cfg.CurrentTarget].Host)
	}
}

func (cfg *config) target(url, name string, checkURL func(*config) bool) {
	if url == "" {
		cfg.printTarget("current")
		return
	}

	// if url is already a key and no name is given, assume url is a name
	if cfg.Targets[url].Host != "" {
		cfg.CurrentTarget = url
		if cfg.save() {
			cfg.printTarget("new")
		}
		return
	}

	if !strings.HasPrefix(url, "http:") && !strings.HasPrefix(url, "https:") {
		url = "https://" + url
	}

	// if an existing target uses url and no name is given, just set it (no check)
	if name == "" {
		for k, v := range cfg.Targets {
			if v.Host == url {
				cfg.CurrentTarget = k
				if cfg.save() {
					cfg.printTarget("new")
				}
				return
			}
		}
	}

	if name != "" {
		cfg.CurrentTarget = name
	} else {
		// didn't specify a target name, make one up.
		for i := 0; ; i++ {
			k := fmt.Sprintf("%v", i)
			if cfg.Targets[k].Host == "" {
				cfg.CurrentTarget = k
				break
			}
		}
	}
	cfg.Targets[cfg.CurrentTarget] = target{Host: url}
	if (checkURL == nil || checkURL(cfg)) && cfg.save() {
		cfg.printTarget("new")
	}
}

func (cfg *config) targets() {
	for k, v := range cfg.Targets {
		cfg.log.info("name: %s\nhost: %s\n\n", k, v.Host)
	}
	cfg.printTarget("current")
}
