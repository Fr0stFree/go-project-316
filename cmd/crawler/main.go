// Package main is a package that provides functionality for crawling web pages and extracting information from them.
package main

import (
	"context"
	"fmt"
	"os"

	"crawler/internal/cliapp"
)

func main() {
	app := cliapp.New()
	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Something went wrong: %v\n", err)
		os.Exit(1)
	}
}
