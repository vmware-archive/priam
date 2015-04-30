package main

import (
	"fmt"
	"github.com/cloudfoundry/cli/plugin"
)

type CfWks struct{}

func (c *CfWks) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name:    "cf-wks",
		Version: plugin.VersionType{Major: 0, Minor: 0, Build: 1},
		Commands: []plugin.Command{
			{
				Name:     "publish",
				HelpText: "push application and publish to Workspace",
				UsageDetails: plugin.Usage{
					Usage: "publish",
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
	if args[0] == "publish" {
		c.Publish()
	} else if args[0] == "unpublish" {
		c.Unpublish()
	}
}

func (c *CfWks) Publish() {
	fmt.Println("Function publish in plugin 'CfWks' is called.")
}

func (c *CfWks) Unpublish() {
	fmt.Println("Function unpublish in plugin 'CfWks' is called.")
}
