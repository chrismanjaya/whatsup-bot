# WhatsUp

A personal WhatsApp bot that tracks income and expenses through natural chat messages — no dedicated app, no manual spreadsheet entry. Send a message like `spent 50k on lunch`, and it's parsed and recorded automatically.

## How it works

1. You send a WhatsApp message describing a transaction (income or expense).
2. The bot (running on a self-hosted Go service) receives it via [whatsmeow](https://github.com/tulir/whatsmeow), a WhatsApp multi-device protocol library.
3. The message is parsed into structured data (type, amount, category) using Google's Gemini API.
4. The transaction is stored in a local SQLite database for fast queries (recent transactions, summaries, splits).
5. On request, data can be exported to Google Sheets for a shareable, human-readable report.

## Features

- Personal expense/income tracking via natural WhatsApp messages (English or Indonesian)
- Registration gate — only registered senders can use the bot, preventing token abuse
- Group chat support — track shared/household expenses with automatic cost-splitting by member
- Fast local queries (SQLite) for things like "last 5 days" without network latency
- On-demand Google Sheets export for a downloadable report
- No web frontend, no external database server — runs as a single Go binary

## Architecture

Built with clean architecture principles — business logic is independent of WhatsApp, Gemini, and storage specifics, making each piece swappable and testable in isolation.

* whatsup-bot/
   * cmd/
      * bot/ # entry point, dependency wiring
   * internal/
      * domain/ # entities (Transaction, User, Group) — no external deps
      * usecase/ # business logic — depends only on domain + port interfaces
      * port/ # interfaces (repositories, parser, exporter)
      * adapter/
      * whatsapp/ # whatsmeow integration
      * gemini/ # LLM-based message parsing
      * sqlite/ # persistence layer
      * sheets/ # Google Sheets export
   * go.mod
   * Makefile
   * .github/
      * workflows/ # CI/CD (auto-deploy on merge to main)

## Tech stack

| Layer | Choice |
|---|---|
| Language | Go |
| WhatsApp integration | [whatsmeow](https://github.com/tulir/whatsmeow) |
| Message parsing | Google Gemini API (free tier) |
| Storage | SQLite (embedded, no separate DB server) |
| Reporting export | Google Sheets API |
| Hosting | GCP `e2-micro` (Always Free tier) |
| Deployment | GitHub Actions → SSH deploy on merge to `main` |

## Prerequisites

- Go 1.21+
- A Gemini API key ([Google AI Studio](https://aistudio.google.com/apikey))
- A Google Cloud service account with access to a target Google Sheet
- A WhatsApp account to link as the bot

## Setup

1. Clone the repo and install dependencies:
```bash
   go mod tidy
```

2. Set environment variables:
```bash
   export GEMINI_API_KEY="your-key-here"
   export SHEETS_SPREADSHEET_ID="your-sheet-id"
   export SHEETS_CREDENTIALS_PATH="./service-account.json"
```

3. Run locally to pair your WhatsApp account:
```bash
   make run
```
   Scan the QR code shown in the terminal with WhatsApp → Linked Devices.

4. Send yourself a test message: `spent 50k on lunch`

## Deployment

See `Makefile` for available commands:

```bash
make build-linux   # cross-compile for the GCP VM
make deploy        # build, copy to VM, restart the service
make logs          # tail live logs on the VM
```

Merges to `main` also trigger an automatic deploy via GitHub Actions (see `.github/workflows/deploy.yml`).

## Usage

| Message | Effect |
|---|---|
| `spent 50k on lunch` | Records an outcome transaction |
| `got salary 15jt` | Records an income transaction |
| `register <name> <email>` | Registers a new user (required before other commands work) |
| `split` (in a group) | Summarizes and splits shared expenses among group members |
| anything unrelated | Politely declined — the bot only handles transactions |

## Security notes

- Only registered WhatsApp numbers can trigger LLM calls (prevents quota abuse).
- API keys and service account credentials are never committed — see `.gitignore`.
- The Gemini free tier's data-usage terms apply to processed messages; see [Google's terms](https://ai.google.dev/gemini-api/terms) for details.

## License

Personal project — not intended for public/commercial use.