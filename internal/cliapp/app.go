// Package cliapp is a package that provides functionality for the command-line interface application.
package cliapp

import (
	"code"
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/urfave/cli/v3"
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
		Value: 10,
		Usage: "crawl depth",
		Validator: func(value int) error {
			if value < 1 {
				return cli.Exit("depth must be greater than or equal to 1", 1)
			}

			return nil
		},
	},
	&cli.IntFlag{
		Name:  "retries",
		Value: 1,
		Usage: "number of retries for failed requests",
		Validator: func(value int) error {
			if value < 0 {
				return cli.Exit("retries must be greater than or equal to 0", 1)
			}

			return nil
		},
	},
	&cli.StringFlag{
		Name:  "delay",
		Usage: "delay between requests (example: 200ms, 1s)",
		Value: "0s",
		// TODO: add validation for delay format
	},
	&cli.StringFlag{
		Name:  "timeout",
		Usage: "per-request timeout",
		Value: "15s",
		// TODO: add validation for timeout format
	},
	&cli.IntFlag{
		Name:  "rps",
		Value: 0,
		Usage: "limit requests per second (overrides delay)",
		Validator: func(value int) error {
			if value < 0 {
				return cli.Exit("rps must be greater than or equal to 0", 1)
			}

			return nil
		},
	},
	&cli.StringFlag{
		Name:  "user-agent",
		Usage: "custom user agent",
		// TODO: do I need to validate user agent format?
	},
	&cli.IntFlag{
		Name:  "workers",
		Value: 4,
		Usage: "number of concurrent workers",
		Validator: func(value int) error {
			if value <= 1 || value > 1000 {
				return cli.Exit("workers must be between 1 and 1000", 1)
			}

			return nil
		},
	},
}

func cliAction(ctx context.Context, cmd *cli.Command) error {
	url := cmd.StringArg("url")

	opts := code.Options{
		URL:         url,
		Depth:       cmd.Int("depth"),
		Retries:     cmd.Int("retries"),
		Delay:       cmd.String("delay"),
		Timeout:     cmd.String("timeout"),
		UserAgent:   cmd.String("user-agent"),
		Concurrency: cmd.Int("workers"),
		IndentJSON:  true,
		HTTPClient: &http.Client{
			Timeout: time.Second * 5,
		},
	}

	report, err := code.Analyze(ctx, opts)
	if err != nil {
		return cli.Exit(fmt.Sprintf("something went wrong during analysis of `%s`: %v", url, err), 1)
	}

	fmt.Println(string(report))

	return err
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
