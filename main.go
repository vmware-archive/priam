package main

import (
	"os"
	"priam/core"
	"path/filepath"
	"fmt"
	"strings"
)

func main() {
	appName := filepath.Base(os.Args[0])
	if strings.HasPrefix(appName, "cf-") {
		defaultCfgFile := filepath.Join(os.Getenv("HOME"), fmt.Sprintf(".%s.yaml", appName[3:]))
		cfplugin(appName, defaultCfgFile)
		return
	}
	core.Priam(os.Args, os.Stdin, os.Stdout, os.Stderr)
}
