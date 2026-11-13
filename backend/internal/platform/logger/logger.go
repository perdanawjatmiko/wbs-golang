package logger

import (
	"log/slog"
	"os"
)

func New(env string) *slog.Logger {
	var h slog.Handler
	if env == "production" {
		h = slog.NewJSONHandler(os.Stdout, nil)
	} else {
		h = slog.NewTextHandler(os.Stdout, nil)
	}
	return slog.New(h).With("environment", env)
}
