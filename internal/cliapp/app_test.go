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
}
