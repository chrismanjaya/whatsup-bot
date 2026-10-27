package util

import (
	"context"
	"log/slog"
	"os"
	"strings"
)

type logCtxKey struct{}

// SetupLogger configures the process-wide default slog logger. Call it once,
// at startup, before anything else logs. format is "json" (production) or
// "text" (local); level is debug, info, warn or error.
func SetupLogger(level, format string) {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)
	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}

// WithLogAttrs returns a context whose logger carries args (e.g. request_id),
// so every log line written for this request can be correlated.
func WithLogAttrs(ctx context.Context, args ...any) context.Context {
	return context.WithValue(ctx, logCtxKey{}, Logger(ctx).With(args...))
}

// Logger returns the request-scoped logger stored in ctx, or the default one.
func Logger(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(logCtxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}

// LogError logs err at Error level under msg, keeping the "error" (and, when
// err carries one, "code") keys consistent across call sites.
func LogError(ctx context.Context, msg string, err error, args ...any) {
	Logger(ctx).ErrorContext(ctx, msg, errArgs(err, args)...)
}

// LogWarn logs err at Warn level under msg, with the same keys as LogError.
func LogWarn(ctx context.Context, msg string, err error, args ...any) {
	Logger(ctx).WarnContext(ctx, msg, errArgs(err, args)...)
}

// Fatal logs err and exits the process. Use it only in main, for
// unrecoverable startup failures.
func Fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	os.Exit(1)
}

// MaskID keeps the last 4 characters of an identifier (phone number, email,
// account number) so logs stay correlatable without storing PII.
func MaskID(id string) string {
	const keep = 4
	if len(id) <= keep {
		return strings.Repeat("*", len(id))
	}
	return strings.Repeat("*", len(id)-keep) + id[len(id)-keep:]
}

func errArgs(err error, args []any) []any {
	args = append(args, "error", err)
	if code, ok := CodeOf(err); ok {
		args = append(args, "code", code.Code)
	}
	return args
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
