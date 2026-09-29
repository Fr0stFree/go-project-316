// Package cliapp is a package that provides functionality for the command-line interface application.
package cliapp

import (
	"github.com/urfave/cli/v3"
)

var cliArgs = []cli.Argument{
	&cli.StringArg{
		Name:     "url",
		Required: true,
	},
}

var cliFlags = []cli.Flag{
	&cli.IntFlag{
		Name:  "depth",
		Value: 10,
		Usage: "crawl depth",
	},
	&cli.IntFlag{
		Name:  "retries",
		Value: 1,
		Usage: "number of retries for failed requests",
	},
	&cli.StringFlag{
		Name:  "delay",
		Usage: "delay between requests (example: 200ms, 1s)",
		Value: "0s",
	},
	&cli.StringFlag{
		Name:  "timeout",
		Usage: "per-request timeout",
		Value: "15s",
	},
	&cli.IntFlag{
		Name:  "rps",
		Value: 0,
		Usage: "limit requests per second (overrides delay)",
	},
	&cli.StringFlag{
		Name:  "user-agent",
		Usage: "custom user agent",
	},
	&cli.IntFlag{
		Name:  "workers",
		Value: 4,
		Usage: "number of concurrent workers",
	},
}

// New creates a new CLI application that runs the crawler command with the specified flags and arguments.
func New() *cli.Command {
	return &cli.Command{
		Name:      "hexlet-go-crawler",
		Usage:     "analyze a website structure",
		Flags:     cliFlags,
		Arguments: cliArgs,
		UsageText: "hexlet-go-crawler [global options] command [command options] <url>",
	}
}
