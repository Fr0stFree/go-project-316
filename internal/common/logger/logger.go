package logger

import (
	"log/slog"
	"os"
)

func Configure() {
	handler := slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	)

	slog.SetDefault(slog.New(handler))
}
