---
name: go-logging
description: Team logging standard for Go services using log/slog — logger setup, levels, structured key names, request-scoped attributes, log-once-at-the-boundary rule, PII redaction, what to log for external calls. Use when adding or reviewing any log statement, configuring the logger, or debugging noisy/duplicated logs.
---

# Logging

## Setup
- `log/slog` only. No `fmt.Println`/`log.Printf` outside `main`'s very first lines.
- One call at startup: `util.SetupLogger(cfg.LogLevel, cfg.LogFormat)`.
  `json` in production (machine-parsed), `text` locally.
- Third-party loggers (e.g. whatsmeow's `waLog`) are adapted to slog or set to
  `WARN` so they don't flood output in a different format.

## The one rule: log once, at the boundary
An error is logged **exactly once**, where it is handled (delivery adapter,
goroutine root, `main`). Layers below *return* errors with context; they do not
log them. Duplicated lines for one failure make incidents harder, not easier.

Exceptions — log in place when the error is **handled there and not returned**:
best-effort side effects (`send typing presence failed`), degraded fallbacks
(`invalid category from parser, defaulting to other`), retries (`attempt` key).

## Levels
| Level | Use for |
|---|---|
| `Debug` | payload shapes, LLM raw output, timings while developing; off in prod |
| `Info` | lifecycle (started, connected, shutdown), one line per handled request/message with outcome, external-call usage (tokens, latency) |
| `Warn` | expected/client failures (4xx codes), retries, degraded fallbacks |
| `Error` | unexpected failures (5xx codes), panics recovered, data loss risk |

At the boundary pick the level from the code: `>= 500` → Error, else Warn.

## Keys
`snake_case`, stable, same name everywhere:

| Key | Meaning |
|---|---|
| `error` | the error value (via `util.LogError/LogWarn`) |
| `code` | `ErrCode.Code` (added automatically by the helpers) |
| `request_id` | correlation id (HTTP header / generated per inbound message) |
| `user_id` | internal numeric id — prefer over phone/email |
| `chat_jid` | masked: `util.MaskID(jid)` |
| `action` | routed action/usecase name |
| `attempt` | retry number |
| `duration_ms` | elapsed time |
| `prompt_tokens`, `response_tokens`, `total_tokens` | LLM usage |

## Request-scoped attributes
Attach correlation data once and reuse it:
```go
ctx = util.WithLogAttrs(ctx, "request_id", id, "user_id", user.ID)
util.Logger(ctx).Info("transaction recorded", "action", "record", "duration_ms", ms)
util.LogError(ctx, "message handling failed", err)
```

## PII & secrets — never log
- Message bodies, free-text user input, LLM prompts/outputs (Debug only, and
  only in non-prod), emails, phone numbers/JIDs unmasked, names.
- API keys, tokens, DSNs with passwords, full config structs.
- Whole domain structs via JSON dumps — log the specific fields you need.

## External calls
One Info line per call: target, latency, outcome; token usage for LLMs. Retries
at Warn with `attempt`. Raw responses only at Debug.

## Messages
Lowercase, past-tense event or "<thing> failed": `"server started"`,
`"send reply failed"`. The message is constant; variable data goes in keys
(so logs can be grouped and searched).
