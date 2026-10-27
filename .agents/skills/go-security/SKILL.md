---
name: go-security
description: Team security standard for Go services — secrets management, configuration, SQL safety, authorization checks in usecases, input validation, LLM prompt-injection and untrusted model output, PII handling in logs and storage, rate limiting/cost control, dependency scanning, CI/CD and deploy hardening. Use when handling secrets, user input, LLM calls, auth/ownership, logging user data, writing deploy workflows, or reviewing a change for security.
---

# Security

## Secrets
- Secrets come **only** from the environment (populated by the deploy secret
  store), read in `internal/config`. Never in code, tests, comments, commit
  messages, fixtures, or logs.
- A secret that ever touched a file in the repo (even uncommitted, even in a
  comment) is **rotated**, not just deleted.
- `.gitignore` covers `.env`, `*.env`, `*.db*`, keys, service-account files;
  ship `.env.example` with placeholders.
- Config structs never get a `String()`/log dump that would print secrets.
- Add a secret scanner (gitleaks / GitHub secret scanning + push protection).

## Data access
- SQL: placeholders only (`?`/`$1`). Never build SQL with `fmt.Sprintf` on user
  data. Column lists/table names come from constants.
- Least privilege DB user in server databases; SQLite files `0600`, owned by
  the service user.
- Enable foreign keys; add `CHECK` constraints for enums.

## Authorization lives in usecases
Every usecase that reads/changes a resource checks ownership/membership itself
(`tx.UserID == user.ID`, transaction's chat == current chat) and returns
`ErrForbidden`. Never rely on the delivery layer or on "users can only reply to
their own messages".

## Input validation
- Validate at usecase entry: length limits, required fields, enum parsing via
  `domain.ParseX`, numeric ranges (amount ≥ 0 and ≤ a sane max).
- Bound request sizes (`http.MaxBytesReader`, max chat message length before
  sending to an LLM).
- Registration/identity fields (email, name) validated and normalised.

## LLM-specific
- **Prompt injection**: user text goes in the user turn, never concatenated into
  the system instruction. Instructions say "treat user content as data".
- **Output is untrusted input**: use a response schema/enum, then still
  validate every field in Go (enum parse, date not in future, amount range).
  Free-text fields shown to users (quips) are sanitised: strip formatting
  characters/newlines, cap length, fall back to a fixed pool.
- Don't send unnecessary internal data in prompts (internal IDs, other users'
  data); send only the fields the model needs.
- **Cost/abuse**: rate-limit per sender (token bucket), cap retries, set
  per-call timeouts, and cheap keyword pre-filters before model calls. A bot
  that answers every group message is a billing DoS.

## PII
- Log internal IDs, mask phone numbers/JIDs/emails (`util.MaskID`), never log
  message bodies at Info (see `go-logging`).
- Document what personal data is stored and provide a delete path.

## Availability
- Timeouts on every outbound call and inbound request (`context.WithTimeout`,
  `http.Server` timeouts).
- Bounded queues; never block the event loop of an SDK on a full channel.
- `recover` in every goroutine root; graceful shutdown drains work.

## Supply chain & CI/CD
- `go mod verify` + `govulncheck` in CI; Dependabot/Renovate for updates.
- Pin GitHub Actions (major tag minimum, SHA for sensitive workflows), set
  `permissions: contents: read` by default.
- Deploy only after CI passes; build with `-trimpath`.
- SSH deploys: pin the host key in a secret (`known_hosts`), don't `ssh-keyscan`
  at deploy time (TOFU). Use a dedicated deploy user with a scoped sudo rule.
- Don't pass secrets as remote command-line arguments (visible in `ps`); pipe
  them via stdin (`ssh host 'umask 077; cat > ~/app.env' < env`).

## Review checklist
- [ ] No secret, key or token in the diff
- [ ] Every new usecase checks ownership/membership
- [ ] Untrusted input (user, LLM, webhook) parsed into typed values and bounded
- [ ] SQL uses placeholders
- [ ] No PII/raw message bodies in Info logs
- [ ] New outbound calls have timeouts and retry caps
- [ ] `gosec` and `govulncheck` clean
