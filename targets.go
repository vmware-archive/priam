package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"strings"
)

func putTarget(prefix string) {
	if appCfg.CurrentTarget == "" {
		log(linfo, "no target set\n")
	} else {
		log(linfo, "%s target is: %s\nname: %s\n", prefix,
			appCfg.Targets[appCfg.CurrentTarget].Host, appCfg.CurrentTarget)
	}
}

func cmdTarget(c *cli.Context) {
	a := c.Args()
	if len(a) < 1 {
		putTarget("current")
		return
	}

	// if a[0] is a key, use it
	if appCfg.Targets[a[0]].Host != "" {
		appCfg.CurrentTarget = a[0]
		putAppConfig()
		putTarget("new")
		return

	}

	if !strings.HasPrefix(a[0], "http:") && !strings.HasPrefix(a[0], "https:") {
		a[0] = "https://" + a[0]
	}

	// if an existing target uses this host a[0], set it
	reuseTarget := ""
	if len(a) < 2 {
		for k, v := range appCfg.Targets {
			if v.Host == a[0] {
				reuseTarget = k
				break
			}
		}
	}

	if reuseTarget != "" {
		appCfg.CurrentTarget = reuseTarget
		putAppConfig()
		putTarget("new")
		return
	}

	if len(a) > 1 {
		appCfg.CurrentTarget = a[1]
	} else {
		// didn't specify a target name, make one up.
		for i := 0; ; i++ {
			k := fmt.Sprintf("%v", i)
			if appCfg.Targets[k].Host == "" {
				appCfg.CurrentTarget = k
				break
			}
		}
	}
	appCfg.Targets[appCfg.CurrentTarget] = target{Host: a[0]}
	if !c.Bool("force") {
		body, err := checkHealth()
		if err != nil {
			log(lerr, "Error checking health of %s: \n", a[0])
			return
		}
		log(ldebug, "health output:\n%s\n", string(body))
		if !strings.Contains(string(body), "allOk") {
			log(lerr, "Reply from %s does not look like Workspace\n", a[0])
			return
		}
	}
	putAppConfig()
	putTarget("new")
}

func cmdTargets(c *cli.Context) {
	for k, v := range appCfg.Targets {
		log(linfo, "name: %s\nhost: %s\n\n", k, v.Host)
	}
	putTarget("current")
}
