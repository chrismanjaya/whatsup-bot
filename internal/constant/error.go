package constant

import (
	"strconv"
)

// ErrCode is a stable, classifiable application error: a numeric Code (HTTP
// status semantics, so it reads the same to anyone familiar with them) plus
// a developer-facing Message. Show only the Code to users — enough for them
// to report an issue without exposing internal detail — and keep the full
// error, Message included, in logs. Every error built from one of these
// (e.g. util.Wrap(ErrNotFound, "user not registered")), however much extra
// context gets wrapped around it, still satisfies errors.Is(err, ErrNotFound)
// — callers classify failures by code instead of matching on error strings.
type ErrCode struct {
	Code    int
	Message string
}

func (e *ErrCode) Error() string { return strconv.Itoa(e.Code) + " " + e.Message }

var (
	// common errors
	ErrInvalidRequest     = &ErrCode{Code: 400, Message: "invalid request"}
	ErrInvalidGroup       = &ErrCode{Code: 401, Message: "invalid group"}
	ErrInvalidUser        = &ErrCode{Code: 402, Message: "invalid user"}
	ErrUserNotRegister    = &ErrCode{Code: 403, Message: "user not register"}
	ErrAlreadyRegister    = &ErrCode{Code: 409, Message: "already register"}
	ErrTooManyRequests    = &ErrCode{Code: 429, Message: "too many requests"}
	ErrServiceUnavailable = &ErrCode{Code: 503, Message: "service unavailable"}
	ErrInternal           = &ErrCode{Code: 500, Message: "internal error"}

	// report errors
	ErrReportNotFound           = &ErrCode{Code: 20001, Message: "report not found"}
	ErrUnclearReportPeriod      = &ErrCode{Code: 20002, Message: "report period is unclear"}
	ErrInvalidReportPeriodRange = &ErrCode{Code: 20003, Message: "invalid report period range"}

	// transaction errors
	ErrTransactionNotFound      = &ErrCode{Code: 10001, Message: "reply transaction not found"}
	ErrUnclearTransactionDate   = &ErrCode{Code: 10002, Message: "transaction date is unclear"}
	ErrInvalidTransactionType   = &ErrCode{Code: 10003, Message: "invalid transaction type"}
	ErrInvalidTransactionAmount = &ErrCode{Code: 10004, Message: "invalid transaction amount"}
	ErrFutureTransactionDate    = &ErrCode{Code: 10005, Message: "transaction date is in the future"}
	ErrTransactionUnparsed      = &ErrCode{Code: 10006, Message: "transaction missing from parse result"}

	// integration errors
	ErrPrivateChatOnly = &ErrCode{Code: 30001, Message: "private chat only"}
	ErrInvalidMCPToken = &ErrCode{Code: 30002, Message: "invalid mcp token"}
)
