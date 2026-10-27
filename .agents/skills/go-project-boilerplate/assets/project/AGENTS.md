# Agent guide

Read the skills in `.agents/skills/` before changing code; they are the
team standard and override habits from other projects:

- `go-architecture` — layers, dependency rule, where new code goes
- `go-naming-standard`, `go-code-style` — names, files, formatting
- `go-error-handling`, `go-logging` — error codes, wrapping, log once
- `go-documentation`, `go-lint`, `go-security`

Before you finish a change: `make check` must pass, and any new behavior has
a table-driven test next to the usecase it belongs to.

## Project-specific notes
<!-- Decisions, gotchas and external constraints that the code can't show. -->
