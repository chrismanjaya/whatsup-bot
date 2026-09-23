package constant

import "strconv"

// ErrCode is a stable, classifiable application error: a numeric Code (HTTP
// status semantics, so it reads the same to anyone familiar with them) plus
// a developer-facing Message. Show only the Code to users — enough for them
// to report an issue without exposing internal detail — and keep the full
// error, Message included, in logs. Every error built from one of these
// (e.g. utils.Wrap(ErrNotFound, "user not registered")), however much extra
// context gets wrapped around it, still satisfies errors.Is(err, ErrNotFound)
// — callers classify failures by code instead of matching on error strings.
type ErrCode struct {
	Code    int
	Message string
}

func (e *ErrCode) Error() string { return strconv.Itoa(e.Code) + " " + e.Message }

var (
	ErrInvalidRequest     = &ErrCode{Code: 400, Message: "invalid request"}
	ErrNotFound           = &ErrCode{Code: 404, Message: "not found"}
	ErrAlreadyExists      = &ErrCode{Code: 409, Message: "already exists"}
	ErrParseFailed        = &ErrCode{Code: 422, Message: "failed to parse"}
	ErrServiceUnavailable = &ErrCode{Code: 503, Message: "service unavailable"}
	ErrInternal           = &ErrCode{Code: 500, Message: "internal error"}
)
