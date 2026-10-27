---
name: go-naming-standard
description: Team naming standard for Go code — packages, files, import aliases, types, usecases, repositories, ports, constants/enums, errors, receivers, SQL query constants, env vars and log keys. Use whenever you create or rename an identifier, file or package, or review naming in a diff.
---

# Naming standard

Go's own conventions (Effective Go, Go Code Review Comments) apply first. These
rules add the team-specific parts.

## Packages & files
| Thing | Rule | Example |
|---|---|---|
| Package | lowercase, one word, singular, no `_` | `transaction`, `user`, `sqldb` |
| Feature usecase package | the aggregate name | `usecase/transaction` |
| Repository package | same aggregate name under the engine | `infra/storage/sqlite/transaction` |
| File | `snake_case.go`, named for the operation it holds | `find_by_jid.go`, `get_summary.go` |
| Repository file | one exported method per file, file = method in snake_case | `Delete` → `delete.go` |
| Shared helpers in a package | `util.go` (unexported only) | |
| Wiring files | `module.go`, `port.go`, `repository.go` | |
| Tests | `<file>_test.go` next to the code | `record_test.go` |

Avoid `common`, `helpers`, `misc`, `types` packages. `util` is the single
allowed grab-bag and must stay small.

## Import aliases
Needed only when two packages share a name. Format `<name><layer>`, all
lowercase, no underscores or camelCase:
`transactionuc` (usecase), `transactionrepo` (storage), `whatsappdelivery`,
`whatsappclient`, `httpdelivery`. Use the same alias for the same package in
every file.

## Types & functions
| Thing | Rule | Example |
|---|---|---|
| Usecase struct | Verb or VerbNoun, no `Usecase` suffix | `Record`, `Amend`, `GetSummary` |
| Usecase method | always `Execute` | `func (uc *Record) Execute(ctx, ...)` |
| Usecase I/O | `<Op>Input`, `<Op>Result` | `RecordResult`, `CreateInput` |
| Constructor | `New<Type>` | `NewRecord`, `NewRepo`, `NewClient` |
| Module accessor | `<Name>Usecase()` | `RecordUsecase()` |
| Repository struct | `Repository` (package gives context) | `transaction.Repository` |
| Repository methods | `Create`, `Update`, `Delete`, `FindBy<Key>` (one row, `nil,nil` if absent), `List<By…>` (many rows), `Count…`, `Exists…` | `FindByJID`, `ListByPeriod` |
| Port interface | role + `Repository` / role noun; never vendor | `TransactionRepository`, `MessageParser` |
| Client config | `Config` in the client's package | `sqlite.Config` |
| Receivers | short, consistent per type: `uc` usecase, `r` repository, `h` handler, `c` client, `s` service, `m` module | |
| Booleans | `is/has/can/should` prefix | `isGroup`, `hasMore` |
| Acronyms | keep case: `ID`, `JID`, `URL`, `HTTP`, `JSON` | `userID`, `chatJID`, `parseHTTPBody` |

## Constants & enums
- MixedCaps, never `SCREAMING_SNAKE`: `ActionRecord`, not `ACTION_RECORD`.
- Enums are **typed**; prefix values with the type name:
  ```go
  type Action int
  const (
      ActionUnknown Action = iota
      ActionRecord
      ActionAmend
  )
  type TransactionType string
  const (
      TransactionTypeIncome  TransactionType = "CR"
      TransactionTypeExpense TransactionType = "DB"
  )
  ```
- String enums get `String()`, `Valid()`, `Parse<Type>(string) (T, error)` and an
  `All<Types>` slice as single source of truth.
- SQL query constants: `<method>Query` (`findByJIDQuery`), column lists
  `<entity>Columns`.
- Durations carry the unit in the type, not the name: `chatIdle = 5 * time.Minute`.

## Errors
- Coded errors: `Err<Subject><Problem>` — `ErrTransactionNotFound`,
  `ErrReportPeriodUnclear`.
- Error message strings: lowercase, no punctuation, describe what failed:
  `"lookup user failed"`, `"insert transaction failed"`. Pick one suffix style
  (`… failed`) and use it everywhere.

## Config / env / logs
- Env vars: `SCREAMING_SNAKE`, prefixed by subsystem: `GEMINI_API_KEY`, `DB_DSN`,
  `LOG_LEVEL`. Config struct fields mirror them in MixedCaps (`GeminiAPIKey`).
- Log keys: `snake_case`, stable across call sites: `error`, `code`, `user_id`,
  `chat_jid`, `request_id`, `attempt`, `duration_ms` (see `go-logging`).

## Spelling
Typos in identifiers become API. `misspell` runs in lint; also read your names
aloud (`constractUser`, `sumarize`, `trarnsactions` are real examples to avoid).
