// Package constant holds fixed values shared by every layer: application
// error codes and other compile-time constants. It imports nothing from this
// module, so any layer may depend on it.
package constant

import "strconv"

// ErrCode is a stable, classifiable application error: a numeric Code plus a
// developer-facing Message. Show only the Code (or a friendly text chosen by
// it) to end users and keep the full wrapped error in logs. However much
// context gets wrapped around it, errors.Is(err, ErrNotFound) still holds, so
// callers classify failures by code, never by matching error strings.
type ErrCode struct {
	Code    int
	Message string
}

func (e *ErrCode) Error() string { return strconv.Itoa(e.Code) + " " + e.Message }

// Common codes. They reuse HTTP status semantics, so they read the same to
// anyone who knows HTTP and map 1:1 onto an HTTP delivery.
var (
	ErrInvalidRequest     = &ErrCode{Code: 400, Message: "invalid request"}
	ErrUnauthorized       = &ErrCode{Code: 401, Message: "unauthorized"}
	ErrForbidden          = &ErrCode{Code: 403, Message: "forbidden"}
	ErrNotFound           = &ErrCode{Code: 404, Message: "not found"}
	ErrConflict           = &ErrCode{Code: 409, Message: "conflict"}
	ErrTooManyRequests    = &ErrCode{Code: 429, Message: "too many requests"}
	ErrInternal           = &ErrCode{Code: 500, Message: "internal error"}
	ErrServiceUnavailable = &ErrCode{Code: 503, Message: "service unavailable"}
)

// Domain codes use the shape DDNNN: DD is the domain number, NNN the error
// within it. Register every domain number here before using it.
//
//	10 example
var (
	ErrExampleNameTaken     = &ErrCode{Code: 10001, Message: "example name already taken"}
	ErrExampleInvalidStatus = &ErrCode{Code: 10002, Message: "invalid example status"}
)
