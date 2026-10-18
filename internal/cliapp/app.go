// Package cliapp is a package that provides functionality for the command-line interface application.
package cliapp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/urfave/cli/v3"

	"code/crawler"
	"code/internal/common/fmttools"
)

var cliArgs = []cli.Argument{
	&cli.StringArg{
		Name:     "url",
		Required: true,
	},
}

func cliArgsValidator(_ context.Context, cmd *cli.Command) error {
	if cmd.Args().Len() != 1 {
		return cli.Exit(
			fmt.Sprintf("expected 1 argument, got %d", cmd.Args().Len()),
			1,
		)
	}

	return nil
}

var cliFlags = []cli.Flag{
	&cli.IntFlag{
		Name:  "depth",
		Value: 2,
		Usage: "crawl depth",
	},
	&cli.IntFlag{
		Name:  "retries",
		Value: 1,
		Usage: "number of retries for failed requests",
	},
	&cli.DurationFlag{
		Name:  "delay",
		Usage: "delay between requests (example: 200ms, 1s)",
		Value: 0,
	},
	&cli.DurationFlag{
		Name:  "timeout",
		Usage: "per-request timeout",
		Value: 15 * time.Second,
	},
	&cli.IntFlag{
		Name:  "rps",
		Value: 0,
		Usage: "limit requests per second (overrides delay)",
	},
	&cli.StringFlag{
		Name:  "user-agent",
		Usage: "custom user agent",
		Value: "HexletCrawler/1.0",
	},
	&cli.IntFlag{
		Name:  "workers",
		Value: 4,
		Usage: "number of concurrent workers",
	},
	&cli.BoolFlag{
		Name:  "verbose",
		Usage: "enable debug logging",
	},
}

func cliAction(ctx context.Context, cmd *cli.Command) error {
	url := cmd.StringArg("url")

	opts := crawler.Options{
		URL:         url,
		Depth:       cmd.Int("depth"),
		Retries:     cmd.Int("retries"),
		Delay:       cmd.Duration("delay"),
		Timeout:     cmd.Duration("timeout"),
		UserAgent:   cmd.String("user-agent"),
		Concurrency: cmd.Int("workers"),
		IndentJSON:  true,
		Verbose:     cmd.Bool("verbose"),
	}

	report, err := crawler.Analyze(ctx, opts)
	if err != nil && !errors.Is(err, context.Canceled) {
		return cli.Exit(fmt.Sprintf("something went wrong during analysis of `%s`: %v", url, err), 1)
	}

	data, err := fmttools.ToJSON(report, opts.IndentJSON)
	if err != nil {
		return cli.Exit(fmt.Sprintf("could not format report: %v", err), 1)
	}

	fmt.Println(string(data))

	return nil
}

// New creates a new CLI application that runs the crawler command with the specified flags and arguments.
func New() *cli.Command {
	return &cli.Command{
		Name:         "hexlet-go-crawler",
		Usage:        "analyze a website structure",
		Arguments:    cliArgs,
		ArgValidator: cliArgsValidator,
		Flags:        cliFlags,
		Action:       cliAction,
		UsageText:    "hexlet-go-crawler [global options] command [command options] <url>",
	}
}
