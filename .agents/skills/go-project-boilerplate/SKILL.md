---
name: go-project-boilerplate
description: Scaffold a new Go service, or retrofit an existing one, with the team's standard base — layered layout (cmd, config, constant, util, domain, usecase, infra), error codes and wrapping, slog logging, config loading, a sample usecase with tests, a SQL repository, an HTTP delivery adapter with graceful shutdown, golangci-lint v2 config with depguard layering, Makefile, CI workflow, .gitignore, .editorconfig, .env.example, AGENTS.md, README and ADR templates. Use when starting a new Go project, or when asked to bring this project's standards/config into another repository.
---

# Go project boilerplate

A runnable, lint-clean skeleton lives in `assets/project/` (module
`example.com/app`). It compiles with only the standard library; you add the
database driver and vendor SDKs you need.

## New project
```bash
SRC=<path-to>/.agents/skills/go-project-boilerplate/assets/project
DST=~/code/my-service
MOD=github.com/acme/my-service

mkdir -p "$DST" && cp -R "$SRC"/. "$DST"/
cd "$DST"
# rename the module everywhere (go.mod, imports, .golangci.yml)
grep -rl 'example.com/app' . | xargs sed -i.bak "s#example.com/app#$MOD#g" && find . -name '*.bak' -delete
# bring the standards along
mkdir -p .agents/skills && cp -R <path-to>/.agents/skills/go-* .agents/skills/
go mod tidy && make tools && make check
```
Then:
1. Pick the DB driver: blank-import it in `cmd/wire.go`, set `DB_DRIVER`/`DB_DSN`
   in `.env` (copy `.env.example`). Placeholders are `?` (SQLite/MySQL); change
   to `$1` for Postgres.
2. Replace the `example` domain/usecase/repository with your first feature
   (keep the shape — see `go-architecture` "Adding a feature").
3. Replace the HTTP delivery if your transport differs (chat bot, queue
   consumer); keep the boundary responsibilities: map input → call usecase →
   log error once → map code → respond.
4. Fill in `README.md` and `AGENTS.md`; add a deploy workflow that `needs:` CI.

## Retrofitting an existing repo
Copy only the tooling first, then converge code over time:
1. `.golangci.yml` (fix module path), `Makefile` targets `lint/vuln/check/tools`,
   `.github/workflows/ci.yml`, `.gitignore`, `.editorconfig`, `.env.example`.
2. Run lint with `--new-from-rev` (see `go-lint`) so the backlog doesn't block.
3. Adopt `internal/constant/error.go` + `internal/util/error.go` (Wrap without
   logging), then `internal/util/log.go` (ctx-aware, MaskID).
4. Move to the layout incrementally, one feature package at a time.

## What's in `assets/project/`
| Path | Purpose |
|---|---|
| `cmd/main.go` | signal-aware context, config → logger → Build → serve → graceful shutdown |
| `cmd/wire.go` | composition root, `App` with ordered closers |
| `internal/config/config.go` | env loading with defaults, aggregated validation errors |
| `internal/constant/error.go` | `ErrCode`, HTTP-semantics common codes, `DDNNN` domain codes |
| `internal/util/error.go` | `Wrap`, `WrapStd` (nil-safe), `CodeOf`, `CodeOrInternal` — no logging |
| `internal/util/log.go` | slog setup (text/json, level), ctx-scoped attrs, `LogError/LogWarn`, `Fatal`, `MaskID` |
| `internal/domain/example.go` | entity + typed enum with `String/Valid/Parse` + `All…` |
| `internal/usecase/example/` | `module.go`, `port.go`, `create.go`, table-driven `create_test.go` |
| `internal/infra/storage/sqldb/` | pool setup with ping/limits, bootstrap schema, one-file-per-method repository |
| `internal/infra/delivery/http/` | handler with error→status mapping, request-id + recover middleware |
| `.golangci.yml` | v2 config incl. depguard layer rules |
| `Makefile` | help, run, build, test, cover, fmt, vet, lint, vuln, tools, tidy, check, clean |
| `.github/workflows/ci.yml` | verify, vet, lint, race tests, govulncheck, build |
| `.gitignore`, `.editorconfig`, `.env.example` | hygiene & secrets safety |
| `AGENTS.md`, `README.md`, `docs/adr/0000-template.md` | documentation starters |

Keep this skeleton green: after editing it, run `go build ./... && go test ./...
&& golangci-lint run ./...` inside `assets/project/`.
