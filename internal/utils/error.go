package utils

import (
	"errors"
	"fmt"
	"log/slog"

	"whatsup-bot/internal/constant"
)

// Wrap adds msg as context to err, in the standard "msg: err" shape used
// throughout the app. err may itself be a constant.ErrCode, for a business
// error with no lower-level cause.
func Wrap(err error, msg string) error {
	return fmt.Errorf("%s: %w", msg, err)
}

// WrapStd classifies cause with a standard error code while keeping cause in
// the chain, so callers can classify the failure with
// errors.Is(err, constant.ErrXxx) without losing the underlying detail.
func WrapStd(code *constant.ErrCode, msg string, cause error) error {
	return fmt.Errorf("%s: %w: %w", msg, code, cause)
}

// CodeOf returns the standard error code carried in err's chain, if any.
func CodeOf(err error) (*constant.ErrCode, bool) {
	var code *constant.ErrCode
	if errors.As(err, &code) {
		return code, true
	}
	return nil, false
}

// Fatal logs err at Error level under msg, then panics. Use it for
// unrecoverable startup failures.
func Fatal(msg string, err error) {
	slog.Error(msg, "error", err)
	panic(err)
}
