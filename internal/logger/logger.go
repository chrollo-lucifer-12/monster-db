package logger

import (
	"log/slog"
	"os"
)

func Init() {
	slog.SetDefault(
		slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelInfo,
			}),
		),
	)
}
