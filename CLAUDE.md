# WhatsUp — WhatsApp Finance Tracker Bot

## Architecture
Clean Architecture: domain -> usecase -> port (interfaces) -> adapter (sqlite, gemini, whatsapp) -> cmd/bot/main.go (composition root, only place importing concrete adapters).

## Conventions
- Enums as typed strings with String()/Valid()/ParseX() constructors (see domain/transaction.go's TransactionType, domain/category.go's Category).
- domain.Category.AllCategories is the single source of truth for valid categories — Valid(), the Gemini schema Enum, and the prompt's category list all derive from it. Add/remove categories there only.
- Errors: usecases return sentinel errors from internal/constant, classified into user-facing messages by replyForError() in main.go via errors.Is(). Don't return raw message strings from usecases for expected failure paths.
- Config: single source of truth is GitHub Secrets, written to ~/whatsup-bot.env on the VM by the deploy workflow, loaded via systemd's EnvironmentFile=. Never hardcode secrets or set env vars manually on the VM.

## Known gotchas / decisions made
- Gemini model: use gemini-3.5-flash-lite (or check https://aistudio.google.com/rate-limit for current best Lite-tier model). Non-lite Flash models have a ~20 requests/day free-tier cap — too low for real use. Lite tier gets ~500/day.
- Retry logic distinguishes 503/UNAVAILABLE (retry with backoff) from 429/RESOURCE_EXHAUSTED (quota — fail fast, retrying can't help).
- ThinkingConfig.ThinkingBudget=0 disables reasoning tokens — needed to stop token usage ballooning on non-lite models (lite models don't seem to need this).
- transaction_date (user-stated, can be backdated via "kemarin" etc.) is separate from created_at (audit timestamp, always now()) — don't conflate them.
- Self-echo guard: main.go's event handler checks v.Info.IsFromMe first (the real fix) and handleMessage also still checks reply-text prefixes (e.g. "*EXPENSE*", "*DELETED*") as a belt-and-suspenders check — if IsFromMe ever proves unreliable on some WhatsApp client, the prefix list is the fallback. Update the prefix list if a new outgoing reply template's first line changes.
- No migration tooling — still in sandbox phase, so schema changes just edit initSchemaQuery (CREATE TABLE IF NOT EXISTS) in adapter/sqlite/const.go and assume a fresh DB (delete whatsup.db and let it recreate). Revisit once there's real data worth preserving across schema changes.
- Only the transaction's original recorder can update/delete it via reply (checked by user.ID == tx.UserID in AmendTransactionUseCase) — even for shared/group transactions, other group members can't amend someone else's entry. Revisit if that turns out to be too restrictive for shared expenses.

## Deploy
Push to main -> GitHub Actions builds binary on the runner (not the VM, too slow) -> scp's it -> SSH restarts systemd service via a scoped passwordless sudo rule (systemctl restart whatsup-bot only).

## Update/delete via reply (done)
Reply to a bot transaction confirmation to edit or delete it — no visible transaction ID is shown to the user; matching happens purely through WhatsApp's own quoted-reply mechanism.
- Outgoing confirmations are captured via client.SendMessage's resp.ID and stored on the transaction row (transactions.wa_message_id) through TransactionRepository.SetWAMessageID, called from main.go right after a successful send.
- Incoming replies are detected via v.Message.GetExtendedTextMessage().GetContextInfo().GetStanzaID() in main.go's event handler and passed into handleMessage as stanzaID; when non-empty, it's tried against AmendTransactionUseCase before falling through to the normal register/split/record flow.
- AmendTransactionUseCase (internal/usecase/amend_transaction.go) looks up the transaction by wa_message_id, checks ownership, then calls MessageParser.ParseAmend (Gemini) with the transaction's current fields to decide action: "update" | "delete" | "none". On update it re-renders the same *EXPENSE*/*INCOME* confirmation template (so a reply to that follow-up also resolves correctly once its own wa_message_id is set), on delete it sends a *DELETED* confirmation, and "none" asks the user to clarify.
- If stanzaID doesn't match any stored wa_message_id (e.g. reply to an unrelated bot message), AmendTransactionUseCase reports handled=false and main.go falls back to normal handling instead of erroring.

## In progress
Nothing currently tracked here.