---
name: go-lint
description: Team lint and quality-gate standard — golangci-lint v2 configuration (correctness, security, depguard layer rules, style), formatting with gofmt/goimports, go vet, race tests, govulncheck, Makefile targets and CI workflow. Use when setting up linting in a project, fixing lint failures, adding nolint directives, or changing CI quality gates.
---

# Lint & quality gates

## Gates (local == CI)
`make check` runs, in order: `fmt` → `vet` → `lint` → `test` (`-race`) → `vuln`.
CI (`.github/workflows/ci.yml`) runs the same on every PR and on `main`;
**deploy workflows depend on CI passing** (`needs:` or branch protection).

## Config
Copy `../go-project-boilerplate/assets/project/.golangci.yml` and replace
`example.com/app` with your module path (two places: `depguard` rules and
`goimports.local-prefixes`). It is golangci-lint **v2** format, pinned to
`v2.12.2` in the Makefile and CI.

Enabled linters and why:
| Group | Linters |
|---|---|
| correctness | errcheck, govet (all but fieldalignment/shadow), staticcheck, ineffassign, unused, nilerr, errorlint, exhaustive |
| database/http | rowserrcheck, sqlclosecheck, bodyclose, noctx |
| security | gosec |
| architecture | depguard — enforces the onion layering from `go-architecture` |
| hygiene | revive (exported docs, naming, error strings, context-first), misspell, unconvert, unparam, gocritic (diagnostic), nolintlint |
| formatters | gofmt, goimports with local prefix (3 import groups) |

Deliberately **off** (too noisy for the value): `contextcheck`, `errname` (our
`ErrCode` type predates the `XxxError` convention), gocritic `performance` tag
(`hugeParam` on config structs), `wrapcheck`, `varnamelen`, `exhaustruct`.

## depguard layer rules
```yaml
domain:   deny usecase, infra, util
usecase:  deny infra, config, database/sql, net/http
shared:   (constant, util) deny usecase, infra
```
Add a rule when you add a layer. Note: blank imports (`_ "pkg"`) are not
reported under the default exclusion presets — test rules with a real import.

## nolint policy
`//nolint:<linter> // reason` — specific linter **and** a reason, enforced by
`nolintlint`. No file- or package-wide disables. Prefer fixing.

## Adopting in an existing repo
1. Add the config, run `golangci-lint run --new-from-rev=origin/main` in CI so
   only new code must be clean.
2. Burn down the backlog per linter, fixing correctness/security first.
3. Drop `--new-from-rev` once the count is zero.

## Commands
```bash
make tools                       # install pinned golangci-lint
golangci-lint run ./...          # lint
golangci-lint fmt ./...          # format (gofmt + goimports)
golangci-lint run --fix ./...    # auto-fix what is safe
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```
