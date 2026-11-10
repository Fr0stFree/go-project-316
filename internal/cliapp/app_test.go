package cliapp_test

import (
	"context"
	"testing"

	"code/internal/cliapp"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestNew(t *testing.T) {
	t.Run("missing URL", func(t *testing.T) {
		command := cliapp.New()
		command.ExitErrHandler = func(context.Context, *cli.Command, error) {}

		err := command.Run(context.Background(), []string{"hexlet-go-crawler"})

		require.Error(t, err)
	})

	t.Run("extra URL", func(t *testing.T) {
		command := cliapp.New()
		command.ExitErrHandler = func(context.Context, *cli.Command, error) {}

		err := command.Run(context.Background(), []string{
			"hexlet-go-crawler", "https://example.com", "https://other.com",
		})

		require.ErrorContains(t, err, "expected 1 argument, got 2")
	})

	t.Run("zero timeout", func(t *testing.T) {
		command := cliapp.New()
		command.ExitErrHandler = func(context.Context, *cli.Command, error) {}

		err := command.Run(context.Background(), []string{
			"hexlet-go-crawler",
			"--timeout", "0s",
			"https://example.com",
		})

		require.ErrorContains(t, err, "--timeout must be greater than zero")
	})

	t.Run("zero workers", func(t *testing.T) {
		command := cliapp.New()
		command.ExitErrHandler = func(context.Context, *cli.Command, error) {}

		err := command.Run(context.Background(), []string{
			"hexlet-go-crawler",
			"--workers", "0",
			"https://example.com",
		})

		require.ErrorContains(t, err, "--workers must be greater than zero")
	})

	t.Run("empty user agent", func(t *testing.T) {
		command := cliapp.New()
		command.ExitErrHandler = func(context.Context, *cli.Command, error) {}

		err := command.Run(context.Background(), []string{
			"hexlet-go-crawler",
			"--user-agent", "",
			"https://example.com",
		})

		require.ErrorContains(t, err, "--user-agent cannot be empty")
	})
}
