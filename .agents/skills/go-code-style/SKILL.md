---
name: go-code-style
description: Team Go code style — formatting, import grouping, function shape, early returns, context usage, constructors and dependency injection, concurrency, generics, testing style. Use when writing or reviewing any Go code in a project that follows this standard.
---

# Go code style

## Formatting
- `gofmt` + `goimports` (via `golangci-lint fmt`). Never hand-format.
- Imports in **three groups**: standard library, third-party, this module.
  `goimports` `local-prefixes` enforces it.
  ```go
  import (
      "context"
      "time"

      "google.golang.org/genai"

      "whatsupbot/internal/constant"
      "whatsupbot/internal/domain"
  )
  ```

## Function shape
- `ctx context.Context` is the first parameter of anything that does I/O.
- Happy path on the left: handle the error, return early, no `else` after `return`.
- Keep `Execute` readable as a list of steps, each introduced by a short
  lowercase comment (`// validate user`, `// parse message`, `// create transaction`).
  Extract a step into an unexported helper when it exceeds ~15 lines.
- More than ~4 parameters → pass an `Input` struct. Positional lists of 7 ints
  are a bug magnet.
- Return concrete types, accept interfaces.
- No naked returns; no named results except for `defer` error handling.
- Prefer `var x T` for zero values, `:=` otherwise; don't spell types the
  compiler infers (`address := fmt.Sprintf(...)`, not `var address string = ...`).

## Structs & constructors
- Dependencies are unexported fields set by `New<Type>(deps...)`; no setters,
  no globals. Constructors validate required deps and return `(*T, error)` when
  setup can fail.
- Constructor parameter order: repositories, external clients, then plain values.
- Do not store `context.Context` in a struct.

## Data handling
- Money is `int64` in the smallest unit; never `float64`.
- Time: store and compare in UTC or one explicit `*time.Location` (e.g.
  `util.Jakarta`); never `time.Local` (servers run in UTC). Inject a
  `now func() time.Time` where tests need a fixed clock.
- Parse external strings into typed enums at the boundary (`domain.ParseX`);
  inside the core, use the typed value.
- Pre-size slices when the length is known: `make([]T, 0, len(src))`.

## Concurrency
- Every goroutine: owned lifecycle, `defer recover()` that logs, exits on
  `ctx.Done()`. Never hold a mutex while blocking on a channel send.
- Sleep with `select { case <-time.After(d): case <-ctx.Done(): return ctx.Err() }`
  so retries and pauses respect cancellation.
- Run tests with `-race` (the Makefile does).

## Generics
Use only when it removes real duplication across types (`FormatNumber[T int|int64]`).
Don't introduce type parameters for one caller.

## Testing
- Table-driven tests with `t.Run(tt.name, ...)`; name cases by behaviour
  (`"rejects taken name"`).
- Fakes for ports are small structs in the `_test.go` file; no mocking framework
  needed for narrow interfaces.
- Assert error classification with `errors.Is(err, constant.ErrX)`, never by
  comparing strings.
- Usecases: unit tests with fakes. Repositories: integration tests against a
  real (temp-file or in-memory) database. Delivery: `httptest` / fake client.
- Pure helpers (parsers, formatters, date math, pagination) always get tests;
  they are where off-by-one bugs live.

## Dead code
Delete it. Unused params, unused fields (`ctx` stored but never read),
computed-but-unused results (`IsFirstToday` with no consumer) and copy-pasted
helpers across packages are review blockers — `unused`/`unparam` catch most.
