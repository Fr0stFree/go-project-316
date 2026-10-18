// Package logger provides configuration for application-wide logging.
package logger

import (
	"log/slog"
	"os"
)

// Configure initializes the default logger with optional debug logging.
func Configure(verbose bool) {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(
		os.Stderr,
		&slog.HandlerOptions{
			Level: level,
		},
	)

	slog.SetDefault(slog.New(handler))
}
