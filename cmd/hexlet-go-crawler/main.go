// Package main is a package that provides functionality for crawling web pages and extracting information from them.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code/internal/cliapp"

	"github.com/urfave/cli/v3"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	if err := cliapp.New().Run(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)

		code := 1

		var exitErr cli.ExitCoder
		if errors.As(err, &exitErr) {
			code = exitErr.ExitCode()
		}

		os.Exit(code)
	}
}
