# WhatsUp — WhatsApp Finance Tracker Bot

## Architecture
Clean Architecture: domain -> usecase -> port (interfaces) -> adapter (sqlite, gemini, whatsapp) -> cmd/bot/main.go (composition root, only place importing concrete adapters).

cmd/bot is the composition root and the only place importing concrete adapters: main.go (load config, connect, wait for signal, disconnect) and wire.go (build() constructs adapters + usecases and registers the WhatsApp handler). Env is read only in internal/config. WhatsApp-specific work lives in internal/adapter/whatsapp: client.go (store/client setup, QR pairing + connect), handler.go (event parsing, group/membership ensure, typing presence, sending replies, storing wa_message_id) and router.go (picks the usecase for a message, plus replyForError). If routing grows, it can be promoted to a usecase since it has no whatsmeow dependency.

## Conventions
- Enums as typed strings with String()/Valid()/ParseX() constructors (see domain/transaction.go's TransactionType, domain/category.go's Category).
- domain.Category.AllCategories is the single source of truth for valid categories — Valid(), the Gemini schema Enum, and the prompt's category list all derive from it. Add/remove categories there only.
- Errors: usecases return sentinel errors from internal/constant, classified into user-facing messages by replyForError() in internal/adapter/whatsapp/router.go via errors.Is(), with the wording taken from internal/message. Don't return raw message strings from usecases for expected failure paths.
- User-facing text: every reply string/template (errors, welcome, split, confirmations) lives in internal/message/message.go — edit wording there, don't inline strings in usecases or the router. BotReplyPrefixes in that file drives the router's self-echo guard, so a new reply template's first line must be added there.
- Config: read via internal/config.Load() (add new env vars there, with defaults). Single source of truth is GitHub Secrets, written to ~/whatsup-bot.env on the VM by the deploy workflow, loaded via systemd's EnvironmentFile=. Never hardcode secrets or set env vars manually on the VM.

## Known gotchas / decisions made
- Gemini model: internal/config's default (when GEMINI_MODEL is unset) is gemini-3.6-flash — a deliberate choice, keep it. Override via the GEMINI_MODEL secret. If quota becomes a problem, use gemini-3.5-flash-lite (or check https://aistudio.google.com/rate-limit for current best Lite-tier model). Non-lite Flash models have a ~20 requests/day free-tier cap — too low for real use. Lite tier gets ~500/day.
- Retry logic distinguishes 503/UNAVAILABLE (retry with backoff) from 429/RESOURCE_EXHAUSTED (quota — fail fast, retrying can't help).
- ThinkingConfig.ThinkingBudget=0 disables reasoning tokens — needed to stop token usage ballooning on non-lite models (lite models don't seem to need this).
- transaction_date (user-stated, can be backdated via "kemarin" etc.) is separate from created_at (audit timestamp, always now()) — don't conflate them.
- Self-echo guard: the whatsapp Handler's handleEvent (handler.go) checks v.Info.IsFromMe first (the real fix) and router.handle (router.go) also still checks reply-text prefixes (e.g. "*EXPENSE*", "*DELETED*") as a belt-and-suspenders check — if IsFromMe ever proves unreliable on some WhatsApp client, the prefix list is the fallback. Update message.BotReplyPrefixes (internal/message/message.go) if a new outgoing reply template's first line changes.
- No migration tooling — still in sandbox phase, so schema changes just edit initSchemaQuery (CREATE TABLE IF NOT EXISTS) in adapter/sqlite/const.go and assume a fresh DB (delete whatsup.db and let it recreate). Revisit once there's real data worth preserving across schema changes.
- Only the transaction's original recorder can update/delete it via reply (checked by user.ID == tx.UserID in AmendTransactionUseCase) — even for shared/group transactions, other group members can't amend someone else's entry. Revisit if that turns out to be too restrictive for shared expenses.

## Deploy
Push to main -> GitHub Actions builds binary on the runner (not the VM, too slow) -> scp's it -> SSH restarts systemd service via a scoped passwordless sudo rule (systemctl restart whatsup-bot only).

## Update/delete via reply (done)
Reply to a bot transaction confirmation to edit or delete it — no visible transaction ID is shown to the user; matching happens purely through WhatsApp's own quoted-reply mechanism.
- Outgoing confirmations are captured via client.SendMessage's resp.ID and stored on the transaction row (transactions.wa_message_id) through TransactionRepository.SetWAMessageID, called from the whatsapp Handler right after a successful send.
- Incoming replies are detected via v.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID() in the whatsapp Handler's handleEvent and passed into router.handle as stanzaID; when non-empty, it's tried against AmendTransactionUseCase before falling through to the normal register/split/record flow.
- AmendTransactionUseCase (internal/usecase/amend_transaction.go) looks up the transaction by wa_message_id, checks ownership, then calls MessageParser.ParseAmend (Gemini) with the transaction's current fields to decide action: "update" | "delete" | "none". On update it re-renders the same *EXPENSE*/*INCOME* confirmation template (so a reply to that follow-up also resolves correctly once its own wa_message_id is set), on delete it sends a *DELETED* confirmation, and "none" asks the user to clarify.
- If stanzaID doesn't match any stored wa_message_id (e.g. reply to an unrelated bot message), AmendTransactionUseCase reports handled=false and the router falls back to normal handling instead of erroring.

## Query transactions with paging (done)
"transaksi tanggal 1 agustus", "transaksi kemarin", "transaksi bulan agustus" list the sender's own transactions (by transaction_date, oldest first), one WhatsApp message per transaction in the normal *EXPENSE*/*INCOME* format, so each can be replied to for update/delete.
- QueryTransactionsUseCase pre-filters on keywords (queryKeywords) before calling MessageParser.ParseQuery (Gemini), so ordinary transaction messages don't cost an extra model call. It runs after register/split and before RecordTransactionUseCase; handled=false falls through to recording.
- 10 transactions per page (queryPageSize). When there are more pages, a *PAGE n/N* summary message follows; its wa_message_id is stored in query_pages (with the date range and page). Replying to it ("next"/"lanjut", "prev"/"sebelumnya", "page 3") is resolved by PageTransactionsUseCase, tried in the router right after AmendTransactionUseCase. Command parsing is plain Go (parsePageCommand), no Gemini call.
- Usecases return []usecase.Reply (Text + optional Tx/Page to link the sent message ID to); the whatsapp handler sends them in order with a short sendInterval pause. Date filtering compares substr(transaction_date,1,10) as text because stored timestamps carry a UTC offset.

## Summary report (done)
"summarize this month", "this week report", "laporan bulan agustus" reply with one *SUMMARY* message: expense/income totals and counts, most used category and biggest transaction per side, then every transaction of the period (date, +/- sign, amount, desc).
- SummarizeTransactionsUseCase (internal/usecase/summarize_transactions.go) pre-filters on summaryKeywords, then MessageParser.ParseSummary (Gemini) resolves the date range. It runs in the router after register/split and before QueryTransactionsUseCase; handled=false falls through.
- Max range is 1 month (inclusive, so 1 Aug–31 Aug ok, 1 Aug–1 Sep too long) -> message.SummaryRangeTooLong. Transactions with amount 0 (still awaiting an amount) are left out. A side with no transactions renders as *none* and its insights are dropped; no transactions at all reuses message.QueryNoResults.
- Fetches everything via ListByUserBetween with limit -1 (SQLite: no limit), no paging.

## In progress
Nothing currently tracked here.