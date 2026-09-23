package utils

import (
	"log/slog"
	"os"
)

// SetupLogger configures the process-wide default slog logger. Call this
// once, at startup, before any other logging happens.
func SetupLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}

// LogError logs err at Error level under msg, keeping the "error" (and,
// when err carries one, "code") keys consistent across call sites.
func LogError(msg string, err error, args ...any) {
	slog.Error(msg, errArgs(err, args)...)
}

// LogWarn logs err at Warn level under msg, keeping the "error" (and, when
// err carries one, "code") keys consistent across call sites.
func LogWarn(msg string, err error, args ...any) {
	slog.Warn(msg, errArgs(err, args)...)
}

func errArgs(err error, args []any) []any {
	args = append(args, "error", err)
	if code, ok := CodeOf(err); ok {
		args = append(args, "code", code.Code)
	}
	return args
}
