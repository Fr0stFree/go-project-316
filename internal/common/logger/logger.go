// Package logger provides configuration for application-wide logging.
package logger

import (
	"log/slog"
	"os"
)

// Configure initializes the default logger with a text handler.
func Configure() {
	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	slog.SetDefault(slog.New(handler))
}
