package util

import (
	"errors"
	"fmt"

	"example.com/app/internal/constant"
)

// Wrap adds msg as context to err, in the standard "msg: err" shape. err may
// itself be a *constant.ErrCode, for a business error with no lower-level
// cause. Wrap never logs: errors are logged once, at the delivery boundary.
func Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// WrapStd classifies cause with a standard error code while keeping cause in
// the chain, so callers can use errors.Is(err, constant.ErrXxx) without losing
// the underlying detail. cause may be nil.
func WrapStd(code *constant.ErrCode, msg string, cause error) error {
	if cause == nil {
		return fmt.Errorf("%s: %w", msg, code)
	}
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

// CodeOrInternal returns the code carried in err's chain, or
// constant.ErrInternal when err is unclassified.
func CodeOrInternal(err error) *constant.ErrCode {
	if code, ok := CodeOf(err); ok {
		return code
	}
	return constant.ErrInternal
}
