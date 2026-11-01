package logger_test

import (
	"log/slog"
	"testing"

	"code/internal/common/logger"

	"github.com/stretchr/testify/require"
)

func TestConfigure(t *testing.T) {
	previous := slog.Default()

	t.Cleanup(func() { slog.SetDefault(previous) })

	t.Run("default level", func(t *testing.T) {
		logger.Configure(false)

		require.False(t, slog.Default().Enabled(t.Context(), slog.LevelDebug))
		require.True(t, slog.Default().Enabled(t.Context(), slog.LevelInfo))
	})

	t.Run("verbose level", func(t *testing.T) {
		logger.Configure(true)

		require.True(t, slog.Default().Enabled(t.Context(), slog.LevelDebug))
	})
}
