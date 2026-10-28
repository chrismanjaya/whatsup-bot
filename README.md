# WhatsUp Bot — WhatsApp finance tracker

A personal finance assistant that lives in WhatsApp. Users chat with **Frankie**, a friendly persona who calls them "bos". Frankie records income and expenses from free text in English or Indonesian ("beli kopi 25rb kemarin"), lets users fix or delete an entry by replying to its confirmation, and builds summaries and paginated transaction lists. In group chats it also splits shared expenses.

Built in Go with [whatsmeow](https://github.com/tulir/whatsmeow) (WhatsApp Web protocol), SQLite, and Google Gemini for structured extraction from natural language.

---

## Features

| Feature | How to use it | Notes |
|---|---|---|
| Register | `register <name> <email>` | Required before anything else. |
| Record a transaction | `buy coffee 5K today`, `gaji 10jt`, `makan siang 35rb kemarin` | Gemini extracts type, amount, category, description and date. Relative dates are resolved in Asia/Jakarta. |
| Update / delete | Reply to a transaction confirmation: `actually 15k`, `ubah kategori jadi food`, `hapus` | Only the user who recorded it can change it. |
| Summary report | `summary this month`, `laporan minggu ini`, `rekap agustus` | Expense and income totals, top category, biggest transaction, every line in the period. Max 1 month. |
| Transaction list | `transaksi kemarin`, `list this week`, `history 1–5 agustus` | 10 per page. Each item can be replied to for update/delete. |
| Paging | Reply to the `PAGE n/N` message: `next` / `lanjut`, `prev` / `kembali` | |
| Group split | `split` in a group chat | Equal split of this month's group expenses across members. |
| Persona replies | Automatic | Errors and quips come from Frankie's message pools, without repeating the last variant. |

Supported categories (single source of truth: `internal/domain/category.go`): food, groceries, household supplies, medicine, health, electronics, fashion, hobby, entertainment, transport, utilities, education, housing, tax, insurance, debt, salary, investment, gift, transfer, other.

---

## Architecture

Clean Architecture in onion layers: **each layer imports only the layers closer to the center.** The full standard is in [`.agents/skills/go-architecture`](.agents/skills/go-architecture/SKILL.md).

```
┌─────────────────────────────────────────────────────────────────────┐
│ cmd/bot, cmd/mcp  main.go (lifecycle) · wire.go (composition root)  │
├─────────────────────────────────────────────────────────────────────┤
│ internal/infra/                                                     │
│   delivery/whatsapp   inbound events → per-chat queue → chatbot     │
│   delivery/mcp        MCP over HTTP, per-user bearer token          │
│   service             Chatbot / MCP: route a request to usecases    │
│   external/gemini     MessageParser implementation (LLM)            │
│   external/whatsapp   whatsmeow client, QR pairing, session store   │
│   storage/sqlite      client + user/group/transaction repositories  │
├─────────────────────────────────────────────────────────────────────┤
│ internal/usecase/                                                   │
│   transaction (record, amend, split) · report (summary, detail)     │
│   user (register, validate) · group (validate)                      │
│   prediction (intent routing) · persona (Frankie) · render (replies)│
├─────────────────────────────────────────────────────────────────────┤
│ internal/domain   Transaction, Category, User, Group, ParsedMessage │
└─────────────────────────────────────────────────────────────────────┘
 side packages: internal/constant (error codes) · internal/util (wrap, log,
 format, time) · internal/template (embedded reply templates) · internal/config
```

### Request flow

```
WhatsApp event
  └─ delivery/whatsapp.Handler.handleEvent      filters self-messages
       └─ chatQueue.enqueue(chatJID)             one worker per chat, in order, chats in parallel
            └─ Handler.process
                 ├─ parseIncomingMessage          text, quoted text, stanza ID, group flag
                 ├─ service.Chatbot.ProcessIncomingMessage
                 │    ├─ prediction.PredictAction        keyword routing (no LLM call)
                 │    ├─ user.Validate / group.Validate
                 │    └─ usecase.Execute → render.*Response → []render.Reply
                 │         (errors → persona.Picker → Frankie error message)
                 └─ for each reply: typing pause → SendMessage → store message ID on
                    the transaction (so a later reply can amend it)
```

### Package conventions

- **Usecase** = one struct per business operation with a single `Execute(ctx, …)`. Each feature package has `module.go` (builds its usecases), `port.go` (the interfaces it needs) and one file per operation.
- **Repository** = one subpackage per aggregate under `infra/storage/sqlite/`, with one file per method and the query constant beside it.
- **Errors** are `*constant.ErrCode` values. Wrap them with `util.Wrap` / `util.WrapStd`, classify with `errors.Is`, and turn them into Frankie messages at the boundary. See [`go-error-handling`](.agents/skills/go-error-handling/SKILL.md).
- **User-facing text** lives in `internal/template/*.tmpl` (embedded) and `internal/usecase/persona/pool.go`.

---

## Getting started

### Prerequisites
- Go **1.26+**
- A C toolchain (cgo is required by `mattn/go-sqlite3`)
- A Gemini API key ([Google AI Studio](https://aistudio.google.com/))
- A WhatsApp account to pair as the bot
- `golangci-lint` v2 for `make lint`

### Run locally
```bash
printf 'GEMINI_API_KEY=...\nGEMINI_MODEL=gemini-3.5-flash-lite\n' > .env   # gitignored
make run                # first run prints a QR code: scan it in WhatsApp → Linked devices
```
The WhatsApp session is stored in `whatsmeow.db`, so later runs reconnect without a QR code.

### Configuration

Read only in `internal/config/config.go`. `make` loads `.env` automatically.

| Variable | Required | Default | Description |
|---|---|---|---|
| `GEMINI_API_KEY` | yes | — | Gemini API key. Never commit it. |
| `GEMINI_MODEL` | no | `gemini-3.5-flash-lite` | Lite-tier models have far higher free quotas than Flash. |
| App DB path | — | `whatsup.db` | Hardcoded today (move to `APP_DB_PATH`). |
| WhatsApp store path | — | `whatsmeow.db` | Hardcoded today (move to `WA_STORE_PATH`). |

### Make targets
| Target | Description |
|---|---|
| `make run` | Run the bot |
| `make build` / `make build-linux` | Build to `bin/` (Linux cross-compile needs a cgo cross toolchain) |
| `make test` / `make cover` | Tests with the race detector / coverage report |
| `make fmt` / `make vet` / `make lint` | Format, vet, golangci-lint |
| `make check` | fmt + vet + lint + test |
| `make reset-db` | Delete the app DB (keeps the WhatsApp session) |
| `make reset-session` | Delete the WhatsApp session (forces QR re-login) |

---

## Data model

The schema is created on startup (`internal/infra/storage/sqlite/const.go`, `CREATE TABLE IF NOT EXISTS`). There are no migrations yet: a schema change means editing that file and recreating `whatsup.db`.

| Table | Key columns |
|---|---|
| `users` | `id`, `jid` (unique), `name`, `email`, `register_at` |
| `groups` | `id`, `jid` (unique), `name`, `created_at` |
| `group_members` | `(group_id, user_id)` PK |
| `transactions` | `id`, `user_id`, `type` (`CR`/`DB`), `amount` (int, rupiah), `category`, `is_shared`, `group_id`, `message_id` (unique, the bot's confirmation message), `transaction_date` (user-stated), `created_at` (audit) |

`transaction_date` is when the money moved, as the user stated it (it can be backdated). `created_at` is when the row was written. Don't conflate them.

---

## Error codes

Defined in `internal/constant/error.go`. Users see a Frankie message chosen by code. Logs carry the code and the full error chain.

| Code | Name | Meaning |
|---|---|---|
| 400 | `ErrInvalidRequest` | Message not understood |
| 401 | `ErrInvalidGroup` | Group could not be resolved |
| 402 | `ErrInvalidUser` | User mismatch / not the owner |
| 403 | `ErrUserNotRegister` | Sender must register first |
| 409 | `ErrAlreadyRegister` | Already registered |
| 500 | `ErrInternal` | Unexpected failure |
| 503 | `ErrServiceUnavailable` | Gemini unavailable / rate limited |
| 10001 | `ErrTransactionNotFound` | Replied-to transaction not found |
| 10002 | `ErrUnclearTransactionDate` | Date could not be resolved |
| 10003 | `ErrInvalidTransactionType` | Neither income nor expense |
| 20001 | `ErrReportNotFound` | No transactions in the period |
| 20002 | `ErrUnclearReportPeriod` | Period could not be resolved |
| 20003 | `ErrInvalidReportPeriodRange` | Period longer than 1 month |

---

## Deployment

Two binaries share one SQLite file (`whatsup.db`, WAL mode so the MCP server reads while the bot writes):

| Binary | Source | systemd unit | Listens |
|---|---|---|---|
| `whatsup-bot` | `cmd/bot` | `whatsup-bot` | WhatsApp only |
| `whatsup-bot-mcp` | `cmd/mcp` | `whatsup-bot-mcp` ([`deploy/whatsup-bot-mcp.service`](deploy/whatsup-bot-mcp.service)) | `127.0.0.1:8080`, behind Caddy |

On a push to `main`, `.github/workflows/deploy.yml`:
1. builds both binaries on the GitHub runner,
2. writes `~/whatsup-bot.env` on the GCP VM from GitHub Secrets (`GEMINI_API_KEY`, `GEMINI_MODEL`, `MCP_URL`),
3. copies the bot binary over and restarts the `whatsup-bot` systemd unit, which loads the env through `EnvironmentFile=`,
4. copies the MCP binary, installs the `whatsup-bot-mcp` unit with the same `User=` and `WorkingDirectory=` as `whatsup-bot`, and restarts it.

Required secrets: `GCP_SSH_PRIVATE_KEY`, `GCP_VM_IP`, `GCP_VM_USER`, `GEMINI_API_KEY`, `GEMINI_MODEL`, `MCP_URL` (public URL users connect to, e.g. `https://mcp.example.com/mcp`). Set configuration only through secrets, never by hand on the VM.

One-time Caddy setup on the VM: add the site block from [`deploy/Caddyfile`](deploy/Caddyfile) to `/etc/caddy/Caddyfile` with your domain, then `sudo systemctl reload caddy`.

### MCP access

A registered user sends `mcp` to the bot in a private chat and gets the MCP URL plus a personal bearer token (sending `mcp` again replaces it). Their AI tool connects with `Authorization: Bearer <token>` and can call `get_transaction_list` (period up to 6 months, paged) for that user's transactions only.

---

## Known issues (refactor in progress)

The `refactor/layers` branch is mid-migration to the `infra/` layout. Fix these before merging to `main`:

- `config.Load` must read `GEMINI_API_KEY` from the environment again.
- Most of the test suite was removed during the refactor. Only routing, message parsing and text matching are covered again; usecase and repository tests still need to be restored.
- Group lookup, paging and the transaction-type label have functional bugs (see the review notes for the list).

---

## Engineering standards

Team standards are agent skills in [`.agents/skills/`](.agents/skills). Any coding agent, and any human, should follow them:

| Skill | Covers |
|---|---|
| [`go-architecture`](.agents/skills/go-architecture/SKILL.md) | Layers, dependency rule, where code goes, adding a feature |
| [`go-naming-standard`](.agents/skills/go-naming-standard/SKILL.md) | Packages, files, aliases, types, enums, errors, env vars |
| [`go-code-style`](.agents/skills/go-code-style/SKILL.md) | Formatting, function shape, concurrency, testing |
| [`go-error-handling`](.agents/skills/go-error-handling/SKILL.md) | `ErrCode`, wrapping, classification, log once at the boundary |
| [`go-logging`](.agents/skills/go-logging/SKILL.md) | slog setup, levels, keys, PII redaction |
| [`go-documentation`](.agents/skills/go-documentation/SKILL.md) | README, doc comments, ADRs, AGENTS.md |
| [`go-lint`](.agents/skills/go-lint/SKILL.md) | golangci-lint v2 config, depguard layering, CI gates |
| [`go-security`](.agents/skills/go-security/SKILL.md) | Secrets, authz, LLM prompt injection, PII, CI/CD hardening |
| [`go-project-boilerplate`](.agents/skills/go-project-boilerplate/SKILL.md) | A runnable, lint-clean skeleton for starting or retrofitting a project |
