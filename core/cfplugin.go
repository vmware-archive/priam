package core

import (
	"flag"
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
	"os"
	"strings"
)

type CfPriam struct{ name, defaultConfigFile string }

const publishUsage string = "publish [-n] [-f MANIFEST_PATH]"
const defaultManifest string = "./manifest.yaml"

func (c *CfPriam) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    c.name,
		Version: plugin.VersionType{Major: 0, Minor: 0, Build: 1},
		Commands: []plugin.Command{
			{
				Name:     "publish",
				HelpText: "push application(s) from a manifest and publish to VMware Identity Manager catalog",
				UsageDetails: plugin.Usage{
					Usage: publishUsage,
					Options: map[string]string{
						"f": "Specify manifest file. Default is " + defaultManifest,
						"n": "No push, only publish",
					},
				},
			},
			{
				Name:     "unpublish",
				HelpText: "remove an app from the VMware Identity Manager catalog",
			},
		},
	}
}

func cfplugin(name, defaultConfigFile string) {
	plugin.Start(&CfPriam{name, defaultConfigFile})
}

func (c *CfPriam) Run(cliConnection plugin.CliConnection, args []string) {
	if args[0] == "publish" {
		c.Publish(cliConnection, args[1:])
	} else if args[0] == "unpublish" {
		c.Unpublish(args[1:])
	}
}

func (c *CfPriam) Publish(cliConn plugin.CliConnection, args []string) {
	flagSet := flag.NewFlagSet("publish", flag.ExitOnError)
	nopush := flagSet.Bool("n", false, "don't push app, just publish")
	trace := flagSet.Bool("t", false, "trace IDM requests")
	manifile := flagSet.String("f", defaultManifest, "manifest file")
	if err := flagSet.Parse(args); err != nil {
		fmt.Printf("Error parsing arguments: %v\nUsage: %s\n", err, publishUsage)
		return
	}

	if !*nopush {
		output, err := cliConn.CliCommand("push", "-f", *manifile)
		if err != nil {
			fmt.Printf("Error pushing app: %v\n%s", err, strings.Join(output, "\n"))
			return
		}
		fmt.Println(strings.Join(output, "\n"))
	}

	// when cf execs a plugin it sets stdin and stdout but not stderr
	log := &logr{traceOn: *trace, errw: os.Stdout, outw: os.Stdout}

	if cfg := newAppConfig(log, c.defaultConfigFile); cfg != nil {
		if ctx := initCtx(cfg, true); ctx != nil {
			publishApps(ctx, *manifile)
		}
	}
}

func (c *CfPriam) Unpublish(args []string) {
	fmt.Println("unpublish is not implemented yet.")
}
