---
name: go-architecture
description: Team architecture standard for Go services (onion / clean architecture with domain, usecase, infra layers, ports declared by consumers, a single composition root). Use when adding a feature, package, repository, external client or delivery adapter, when deciding where code belongs, or when reviewing a change for layering violations.
---

# Go service architecture

Onion layers. **Each layer imports only layers closer to the center.** The rule
is enforced by `depguard` in `.golangci.yml` (see `go-lint`).

```
                ┌──────────────────────────────────────────┐
  cmd/          │ composition root: main.go, wire.go       │  knows every concrete type
                ├──────────────────────────────────────────┤
  infra/        │ delivery/  storage/  external/  (adapters)│  implements ports, talks to the world
                ├──────────────────────────────────────────┤
  usecase/<f>/  │ one package per feature, ports in port.go │  business processes
                ├──────────────────────────────────────────┤
  domain/       │ entities, enums, pure rules               │  imports nothing internal
                └──────────────────────────────────────────┘
  side packages any layer may import: constant/ (error codes, fixed values), util/ (wrap, log, helpers)
  config/ is imported by cmd/ only.
```

## Layout

```
cmd/
  main.go            load config → setup logger → Build → run → graceful shutdown
  wire.go            Build(ctx, cfg): clients → repos → external → usecases → delivery
internal/
  config/            the ONLY package that reads os.Getenv
  constant/          ErrCode values, fixed values (imports nothing internal)
  util/              Wrap/WrapStd/CodeOf, logging, formatting (imports constant only)
  domain/            Entity structs, typed enums with String/Valid/Parse
  usecase/<feature>/
    module.go        Module struct + NewModule(ports...) + <Name>Usecase() accessors
    port.go          interfaces THIS package needs (repositories, external clients)
    <operation>.go   one usecase struct per operation, single Execute method
    <operation>_test.go
  infra/
    delivery/<transport>/   inbound: http, whatsapp, grpc, queue consumers
    storage/<engine>/       client.go + one subpackage per aggregate (repository)
    external/<vendor>/      outbound clients: LLMs, payment APIs, messaging SDKs
  template/          embedded user-facing text (go:embed), when the app renders messages
```

## Rules

1. **Domain** holds data + rules needing no I/O. No `context`, no SQL, no JSON tags
   unless the entity genuinely is a wire format.
2. **Usecase = one business operation.** A struct named for the verb (`Record`,
   `Register`, `GetSummary`) with dependencies as interface fields and one
   `Execute(ctx, ...)` method. Return domain types or a `<Op>Result` struct —
   never transport types (HTTP responses, chat replies, rendered text).
3. **Ports are declared by the consumer** (`usecase/<f>/port.go`) and contain only
   the methods that package calls. Do not copy a full repository interface into
   every package; if a usecase never calls `Delete`, the port has no `Delete`.
4. **Name ports by role, not vendor**: `MessageParser`, `PaymentGateway`, not
   `GeminiClient`, `StripeClient`. Swapping vendors must not touch usecases.
5. **Infra implements ports.** Adapters map between external shapes and domain
   types; business decisions never live here. A repository returns `(nil, nil)`
   for "not found" on single-row lookups; the usecase decides if that is an error.
6. **Delivery adapters are thin**: parse the inbound payload → call one
   orchestrator/usecase → render → send. Error-to-response mapping and the single
   error log line happen here (see `go-error-handling`).
7. **Orchestration across usecases** (routing a chat message to the right usecase,
   validating the sender first) is application logic: put it in
   `usecase/<app>/` (e.g. `usecase/chatbot`), not under `infra/`.
8. **Presentation** (templates, formatting replies) belongs to the delivery side
   (`infra/delivery/<transport>/render` or similar), not inside usecases.
9. **cmd/wire.go is the only place that imports concrete infra packages** and the
   only place `config` values are read. Construct in this order: storage clients
   → repositories → external clients → usecase modules → orchestrators → delivery.
10. **No package-level mutable state** (global template caches, global clients).
    Construct it in wire.go and inject it. `util` stays stateless except the
    process logger.
11. **Every goroutine has an owner**: a context for cancellation, a `recover` that
    logs, and a shutdown path that drains it.
12. **Context**: every I/O call takes `ctx` as first parameter; delivery creates a
    per-request ctx with a deadline (`context.WithTimeout`). Never use
    `context.Background()` below `main`.

## Adding a feature (checklist)

1. Entities/enums in `domain/` (typed enum + `AllX` slice + `Parse` func).
2. Error codes in `constant/error.go` under a new domain number (DDNNN).
3. `usecase/<feature>/port.go` → narrow interfaces; `<op>.go` → usecase; `module.go`.
4. Table-driven test with in-file fakes for the ports (`<op>_test.go`).
5. Repository subpackage in `infra/storage/<engine>/<aggregate>/`, one file per method.
6. Wire it in `cmd/wire.go`; expose through a delivery adapter.
7. `make check`.

## Reference
- Runnable skeleton: `../go-project-boilerplate/assets/project/`
- Reference implementation: whatsup-bot (`internal/usecase/transaction`,
  `internal/infra/storage/sqlite/transaction`).
