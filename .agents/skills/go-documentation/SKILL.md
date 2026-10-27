---
name: go-documentation
description: Team documentation standard — README structure, package and exported-identifier doc comments, when to write inline comments, ADRs for decisions, AGENTS.md for agent/onboarding context, keeping docs in sync with code. Use when writing or reviewing docs, comments, README, ADRs, or after a change that alters behavior, config, commands or architecture.
---

# Documentation

## What lives where
| Doc | Content | Audience |
|---|---|---|
| `README.md` | what it is, quick start, config table, architecture overview, commands, deploy, troubleshooting | new contributor / operator |
| `AGENTS.md` (or `CLAUDE.md`) | pointers to `.agents/skills/`, project-specific gotchas and decisions code can't show | coding agents, new teammates |
| `docs/adr/NNNN-title.md` | one architectural decision: context, decision, consequences, alternatives | future maintainers |
| package doc (`// Package x ...`) | the package's responsibility and boundaries | readers of the code |
| doc comments | contract of exported identifiers | callers |
| inline comments | *why* something non-obvious is done | maintainers |

Docs describe the code **as it is**. A change that alters config, commands,
user-visible behaviour or architecture updates the README/AGENTS.md in the same
PR. Stale docs are worse than none.

## Doc comments
- Every exported type, func, const block and package has one, starting with the
  identifier's name: `// Record stores a new transaction parsed from a chat message.`
- State the contract: what it returns on "not found" (`(nil, nil)`), which error
  codes it can return, concurrency safety, units (`amount in rupiah`).
- One package doc per package, in the most central file or `doc.go`.

## Inline comments
- Explain **why**, constraints, and non-obvious invariants:
  `// Sending while holding mu means the worker can't exit between lookup and hand-off.`
- Short step comments inside long `Execute` methods are the house style
  (`// validate user`, `// parse message`) — keep them to 2–4 words.
- Don't narrate the obvious (`// increment i`), don't leave commented-out code,
  and never leave a commented-out secret or `//os.Getenv(...)` behind.
- `TODO(name): …` must reference an owner or issue.

## README template (sections, in order)
1. Title + one-paragraph purpose
2. Features / usage (commands, endpoints, examples)
3. Architecture (diagram + layout + request flow)
4. Configuration table: variable, required, default, description
5. Local development (`make` targets)
6. Testing & quality gates
7. Deployment
8. Data model / migrations
9. Error codes (if user-visible)
10. Troubleshooting / known limitations
11. Links to skills / ADRs

## ADRs
Write one when a decision is costly to reverse or will be questioned later
(storage engine, LLM vendor/model, auth model, "no migrations yet"). Template:
`../go-project-boilerplate/assets/project/docs/adr/0000-template.md`.
Never edit an accepted ADR's decision — supersede it with a new one.
