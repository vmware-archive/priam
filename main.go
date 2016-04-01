package main

import (
	"os"
	"path/filepath"
	"priam/core"
	"strings"
)

func main() {
	appName := filepath.Base(os.Args[0])
	defaultCfgFile := filepath.Join(os.Getenv("HOME"), ".priam.yaml")
	if strings.HasPrefix(appName, "cf-") {
		cfplugin(appName, defaultCfgFile)
	} else {
		core.Priam(os.Args, defaultCfgFile, os.Stdin, os.Stdout, os.Stderr)
	}
}
