---
name: go-error-handling
description: Team error-handling standard for Go — coded application errors (constant.ErrCode), wrapping with util.Wrap/WrapStd, classification with errors.Is/CodeOf, where errors are logged (once, at the boundary), user-facing vs developer-facing messages, retries, panics. Use when returning, wrapping, classifying, logging or rendering errors, or when adding a new error code.
---

# Error handling

## Model
Errors carry **two things**: a *classification* (a stable `*constant.ErrCode`)
and a *chain of context* (`"create transaction failed: insert transaction failed: UNIQUE constraint…"`).

```go
// internal/constant/error.go
type ErrCode struct {
    Code    int
    Message string // developer-facing, never shown verbatim to users
}
func (e *ErrCode) Error() string { return strconv.Itoa(e.Code) + " " + e.Message }
```

### Code scheme
- **Common codes use real HTTP semantics** — 400 invalid request, 401
  unauthenticated, 403 forbidden (not the owner), 404 not found, 409 conflict,
  422 unprocessable, 429 rate limited, 500 internal, 503 dependency unavailable.
  Don't reuse 401/402/403 for unrelated meanings.
- **Domain codes are `DDNNN`**: `DD` = domain number registered in a comment in
  `error.go` (10 transaction, 20 report, …), `NNN` = sequence. Never renumber a
  shipped code; deprecate and add a new one.
- Names: `Err<Subject><Problem>`.

## The helpers (`internal/util/error.go`)
| Call | Use when |
|---|---|
| `util.Wrap(err, "msg")` | add context, keep existing classification; or classify with no cause: `util.Wrap(constant.ErrInvalidRequest, "empty name")` |
| `util.WrapStd(code, "msg", cause)` | classify an unclassified lower-level error (DB, network) while keeping it in the chain. `cause` may be nil. |
| `util.CodeOf(err)` / `util.CodeOrInternal(err)` | read the classification at the boundary |
| `errors.Is(err, constant.ErrX)` | branch on a classification anywhere |

**`Wrap`/`WrapStd` must not log.** Logging inside a wrapper writes the same
failure N times as it bubbles up. See `go-logging`.

## Rules per layer
- **Infra (repositories, clients)**: wrap with context only (`util.Wrap(err,
  "insert transaction failed")`). Translate *driver-specific* conditions into
  domain meaning where the adapter is the only one who can recognise them:
  `sql.ErrNoRows` → `(nil, nil)`; `RowsAffected()==0` → `ErrXNotFound`; vendor
  HTTP 429/503 → `ErrTooManyRequests`/`ErrServiceUnavailable` (use the SDK's
  typed error, e.g. `errors.As(err, &genai.APIError{})`, not string matching).
- **Usecases**: classify. Every error returned from `Execute` carries a code:
  expected business failures with their specific code, unexpected dependency
  failures with `WrapStd(constant.ErrInternal, …, err)`. Don't re-classify an
  error that already has a more specific code — use `Wrap`.
- **Delivery boundary**: the **only** place that (1) logs the error once with
  full chain + code, (2) maps the code to a response (HTTP status, a friendly
  chat message from a pool), (3) hides internals — users see the friendly text
  or `code` + generic message, never `err.Error()`.

## Do / Don't
```go
// ✅ classify a business failure
if tx.UserID != user.ID {
    return nil, util.Wrap(constant.ErrForbidden, "transaction belongs to another user")
}
// ✅ classify an unexpected failure, keep the cause
if err := uc.repo.Create(ctx, tx); err != nil {
    return nil, util.WrapStd(constant.ErrInternal, "create transaction failed", err)
}
// ❌ string matching
if strings.Contains(err.Error(), "not found") { … }
// ❌ swallowing
_ = uc.repo.SetMessage(ctx, id, msgID)        // at least log it at the boundary
// ❌ replacing a precise error with a generic one at the boundary
return s.handleError(ctx, constant.ErrUserNotRegister) // when err was a DB outage
// ❌ fmt.Errorf("... %w", nil) — produces "%!w(<nil>)"; WrapStd handles nil
```

## Validation errors
Validate at the usecase entry (`ErrInvalidRequest` or a specific code). Parsers
of untrusted input (LLM output, request bodies) return typed errors; the caller
decides whether to default (`CategoryOther`) or reject.

## Retries
Retry only idempotent calls and only on retryable classes (503, timeouts; 429
only with the server's backoff hint). Exponential backoff with a cap, a max
attempt count, and `ctx.Done()` respected while waiting. Log each retry at Warn,
the final failure is returned (not logged) for the boundary to log.

## Panics
Never panic for expected errors. `panic` is for programmer bugs; every goroutine
root and request handler has a `recover` that logs `"panic"` + stack and turns
it into an `ErrInternal` response. In `main`, fatal startup errors use
`util.Fatal` (log + `os.Exit(1)`), not `panic`.

## Adding an error code
1. Add `Err<Subject><Problem>` in the right DD block of `constant/error.go`.
2. Add its user-facing text where the boundary renders errors (HTTP mapping or
   the chat message pool) — every code that can reach a user must have text.
3. Add a test asserting `errors.Is(err, constant.ErrNew)` for the path.
