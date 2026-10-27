package util

import (
	"encoding/json"
	"log/slog"
	"os"
)

// SetupLogger configures the process-wide default slog logger. Call this once, at startup, before any other logging happens.
func SetupLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))
}

// LogInfo logs an informational message and forces the input object to print as a single-line JSON string.
func LogInfo(msg string, obj any, args ...any) {
	slog.Info(msg, infoArgs(obj, args)...)
}

// LogError logs err at Error level under msg, keeping the "error" (and, when err carries one, "code") keys consistent across call sites.
func LogError(msg string, err error, args ...any) {
	slog.Error(msg, errArgs(err, args)...)
}

// LogWarn logs err at Warn level under msg, keeping the "error" (and, when err carries one, "code") keys consistent across call sites.
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

// infoArgs safely marshals the object to JSON and appends it to the log arguments.
func infoArgs(obj any, args []any) []any {
	// Skip appending anything if no object was provided
	if obj == nil {
		return args
	}

	jsonData, err := json.Marshal(obj)
	if err != nil {
		// Log the error natively so logging never breaks your app
		return append(args, "data_error", "unmarshalable object")
	}

	return append(args, "data", string(jsonData))
}
