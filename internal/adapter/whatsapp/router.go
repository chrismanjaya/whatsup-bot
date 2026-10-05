package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/message"
	"whatsup-bot/internal/persona"
	"whatsup-bot/internal/usecase"
	"whatsup-bot/internal/utils"
)

// incoming is a WhatsApp message reduced to the plain fields routing needs,
// so the router has no dependency on whatsmeow types.
type incoming struct {
	text       string
	stanzaID   string
	quotedText string
	senderJID  string
	isGroup    bool
	groupID    int64
}

// router decides which usecase handles an incoming message and turns the
// outcome into the reply text.
type router struct {
	registerUser *usecase.RegisterUserUseCase
	recordTx     *usecase.RecordTransactionUseCase
	amendTx      *usecase.AmendTransactionUseCase
	summarizeTx  *usecase.SummarizeTransactionsUseCase
	queryTx      *usecase.QueryTransactionsUseCase
	pageTx       *usecase.PageTransactionsUseCase
	computeSplit *usecase.ComputeSplitUseCase
	persona      *persona.Picker
}

// handle returns the replies to send, in order. A reply's Tx or Page tells
// the handler which row to link the sent message's WhatsApp ID to. Handlers
// are tried in priority order; the first one that claims the message wins,
// and anything unclaimed is recorded as a transaction.
func (r *router) handle(ctx context.Context, in incoming) []usecase.Reply {
	slog.Info("message received", "sender", in.senderJID, "is_group", in.isGroup, "group_id", in.groupID, "text", in.text, "stanza_id", in.stanzaID, "quoted_text", in.quotedText)

	if isBotReply(in.text) {
		return nil
	}
	if replies, ok := r.handleQuotedReply(ctx, in); ok {
		return replies
	}
	if replies, ok := r.handleCommand(ctx, in); ok {
		return replies
	}
	if replies, ok := r.handleReport(ctx, in); ok {
		return replies
	}
	return r.recordTransaction(ctx, in)
}

// handleQuotedReply resolves a reply against the bot message it quotes: a
// transaction confirmation (amend), a *PAGE n/N* summary (paging) or a quip.
// A stanza ID linked to none of them falls through to normal handling.
func (r *router) handleQuotedReply(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	if in.stanzaID == "" {
		return nil, false
	}
	if replies, ok := r.amendTransaction(ctx, in); ok {
		return replies, true
	}
	if replies, ok := r.pageTransactions(ctx, in); ok {
		return replies, true
	}
	if strings.HasPrefix(in.quotedText, message.QuipPrefix) {
		// A quip isn't linked to its transaction (the confirmation is),
		// so point the user there instead of treating this as new input.
		slog.Info("reply to a quip, pointing user to the confirmation", "sender", in.senderJID, "stanza_id", in.stanzaID)
		return single(r.errorReply(message.PoolReplyToQuip, constant.ErrInvalidRequest.Code, in), nil), true
	}
	slog.Warn("reply stanza id not linked to a tracked transaction, falling back to normal handling",
		"sender", in.senderJID, "stanza_id", in.stanzaID, "quoted_text", in.quotedText)
	return nil, false
}

// handleCommand handles the plain-text commands: register and group split.
func (r *router) handleCommand(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	switch {
	case strings.HasPrefix(in.text, "register "):
		return r.register(ctx, in), true
	case in.text == "split" && in.isGroup:
		return r.split(ctx, in), true
	}
	return nil, false
}

// handleReport handles reporting requests: a summary of a period, or a
// paged listing of transactions.
func (r *router) handleReport(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	if replies, ok := r.summarizeTransactions(ctx, in); ok {
		return replies, true
	}
	return r.queryTransactions(ctx, in)
}

// --- transactions ---

func (r *router) recordTransaction(ctx context.Context, in incoming) []usecase.Reply {
	replies, err := r.recordTx.Execute(ctx, in.senderJID, in.text, in.isGroup, in.groupID)
	if err != nil {
		utils.LogError("record transaction failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil)
	}
	slog.Info("transaction processed", "sender", in.senderJID, "is_group", in.isGroup, "messages", len(replies))
	return replies
}

func (r *router) amendTransaction(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	replies, handled, err := r.amendTx.Execute(ctx, in.senderJID, in.stanzaID, in.text)
	if err != nil {
		utils.LogError("amend transaction failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil), true
	}
	if handled {
		slog.Info("transaction amended", "sender", in.senderJID, "stanza_id", in.stanzaID, "messages", len(replies))
	}
	return replies, handled
}

// --- reports ---

func (r *router) summarizeTransactions(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	summary, handled, err := r.summarizeTx.Execute(ctx, in.senderJID, in.text)
	if err != nil {
		utils.LogError("summarize transactions failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil), true
	}
	if !handled {
		return nil, false
	}
	slog.Info("transactions summarized", "sender", in.senderJID)
	return single(summary, nil), true
}

func (r *router) queryTransactions(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	replies, handled, err := r.queryTx.Execute(ctx, in.senderJID, in.text)
	if err != nil {
		utils.LogError("query transactions failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil), true
	}
	if handled {
		slog.Info("transactions listed", "sender", in.senderJID, "messages", len(replies))
	}
	return replies, handled
}

func (r *router) pageTransactions(ctx context.Context, in incoming) ([]usecase.Reply, bool) {
	replies, handled, err := r.pageTx.Execute(ctx, in.senderJID, in.stanzaID, in.text)
	if err != nil {
		utils.LogError("page transactions failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil), true
	}
	if handled {
		slog.Info("transactions paged", "sender", in.senderJID, "stanza_id", in.stanzaID, "messages", len(replies))
	}
	return replies, handled
}

// --- commands ---

func (r *router) register(ctx context.Context, in incoming) []usecase.Reply {
	parts := strings.Fields(in.text)
	if len(parts) < 3 {
		return single(message.RegisterUsage, nil)
	}
	reply, err := r.registerUser.Execute(ctx, in.senderJID, parts[1], parts[2])
	if err != nil {
		utils.LogError("register failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil)
	}
	slog.Info("user registered", "sender", in.senderJID, "name", parts[1])
	return single(reply, nil)
}

func (r *router) split(ctx context.Context, in incoming) []usecase.Reply {
	reply, err := r.computeSplit.Execute(ctx, in.groupID)
	if err != nil {
		utils.LogError("split failed", err, "group_id", in.groupID)
		return single(r.replyForError(err, in), nil)
	}
	slog.Info("split computed", "group_id", in.groupID)
	return single(reply, nil)
}

// single wraps one usecase result as the reply list; tx may be nil.
func single(text string, tx *domain.Transaction) []usecase.Reply {
	if text == "" {
		return nil
	}
	return []usecase.Reply{{Text: text, Tx: tx}}
}

func isBotReply(text string) bool {
	for _, p := range message.BotReplyPrefixes {
		if strings.HasPrefix(text, p) {
			return true
		}
	}
	return false
}

// replyForError maps a usecase's classified error to the message shown to
// the user, keeping that mapping in one place instead of per call site. The
// wording is one of Frankie's pooled variants, in the language the user
// wrote in. The error code's emoji (message.ErrorEmoji) is appended so a
// screenshot identifies the error without exposing any internal detail; the
// code itself is logged, to match up with the sender and time.
func (r *router) replyForError(err error, in incoming) string {
	key := message.PoolGeneric
	switch {
	case errors.Is(err, constant.ErrAlreadyExists):
		key = message.PoolAlreadyRegistered
	case errors.Is(err, constant.ErrNotFound):
		key = message.PoolNotRegistered
	case errors.Is(err, constant.ErrInvalidRequest):
		key = message.PoolInvalidRequest
	case errors.Is(err, constant.ErrServiceUnavailable):
		key = message.PoolServiceUnavailable
	}
	code := constant.ErrInternal.Code
	if c, ok := utils.CodeOf(err); ok {
		code = c.Code
	}
	return r.errorReply(key, code, in)
}

// errorReply renders a Frankie error reply from pool key, ending with the
// emoji for code.
func (r *router) errorReply(key message.PoolKey, code int, in incoming) string {
	lang := message.ReplyLang
	if lang == "" {
		lang = r.persona.Lang(in.senderJID, in.text)
	}
	msg := r.persona.Error(key, in.senderJID, lang)

	emoji, ok := message.ErrorEmoji[code]
	if !ok {
		emoji = message.ErrorEmojiDefault
	}
	slog.Info("error reply sent", "sender", in.senderJID, "code", code, "emoji", emoji, "pool", key)
	return fmt.Sprintf(message.ErrWithCode, msg, emoji)
}
