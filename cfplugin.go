package main

import (
	"flag"
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
	"os"
	"strings"
)

type CfWks struct{}

const publishUsage string = "publish [-n] [-f MANIFEST_PATH]"
const defaultManifest string = "./manifest.yaml"

func (c *CfWks) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "cf-wks",
		Version: plugin.VersionType{Major: 0, Minor: 0, Build: 1},
		Commands: []plugin.Command{
			{
				Name:     "publish",
				HelpText: "push application(s) from a manifest and publish to Workspace",
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
				HelpText: "remove an app from the workspace catalog",
			},
		},
	}
}

func cfplugin() {
	plugin.Start(new(CfWks))
}

func (c *CfWks) Run(cliConnection plugin.CliConnection, args []string) {
	os.Stderr = os.Stdin // when cf execs a plugin it sets stdin and stdout but not stderr
	if strings.ToLower(os.Getenv("CF_TRACE")) == "true" {
		traceMode = true
	}
	if args[0] == "publish" {
		c.Publish(cliConnection, args[1:])
	} else if args[0] == "unpublish" {
		c.Unpublish(args[1:])
	}
}

func (c *CfWks) Publish(cliConn plugin.CliConnection, args []string) {
	flagSet := flag.NewFlagSet("publish", flag.ExitOnError)
	nopush := flagSet.Bool("n", false, "don't push app, just publish")
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
	if authHdr := authHeader(); authHdr != "" {
		publishApps(authHdr, *manifile)
	}
}

func (c *CfWks) Unpublish(args []string) {
	fmt.Println("unpublish is not implemented yet.")
}
