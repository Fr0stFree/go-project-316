// Package main is a package that provides functionality for crawling web pages and extracting information from them.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code/internal/cliapp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	app := cliapp.New()
	if err := app.Run(ctx, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong: %v\n", err) // TODO: hide error, log instead
		os.Exit(1)
	}
}
