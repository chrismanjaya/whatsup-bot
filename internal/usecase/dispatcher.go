package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"whatsup-bot/internal/constant"
	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/usecase/group"
	"whatsup-bot/internal/usecase/message"
	"whatsup-bot/internal/usecase/persona"
	"whatsup-bot/internal/usecase/response"
	"whatsup-bot/internal/usecase/transaction"
	"whatsup-bot/internal/usecase/user"
	"whatsup-bot/internal/util"
)

// Incoming is a chat message reduced to the plain fields routing needs, so
// the dispatcher has no dependency on the chat platform's types.
type Incoming struct {
	Text       string
	StanzaID   string // ID of the message this one replies to, "" if none
	QuotedText string // text of the message this one replies to
	SenderJID  string
	IsGroup    bool
	GroupID    int64
}

// Dispatcher decides which usecase handles an incoming message and turns the
// outcome, including errors, into the replies to send.
type Dispatcher struct {
	registerUser *user.RegisterUseCase
	recordTx     *transaction.RecordUseCase
	amendTx      *transaction.AmendUseCase
	summarizeTx  *transaction.SummarizeUseCase
	queryTx      *transaction.QueryUseCase
	pageTx       *transaction.PageUseCase
	computeSplit *group.ComputeSplitUseCase
	persona      *persona.Picker
}

func NewDispatcher(
	registerUser *user.RegisterUseCase,
	recordTx *transaction.RecordUseCase,
	amendTx *transaction.AmendUseCase,
	summarizeTx *transaction.SummarizeUseCase,
	queryTx *transaction.QueryUseCase,
	pageTx *transaction.PageUseCase,
	computeSplit *group.ComputeSplitUseCase,
	picker *persona.Picker,
) *Dispatcher {
	return &Dispatcher{
		registerUser: registerUser,
		recordTx:     recordTx,
		amendTx:      amendTx,
		summarizeTx:  summarizeTx,
		queryTx:      queryTx,
		pageTx:       pageTx,
		computeSplit: computeSplit,
		persona:      picker,
	}
}

// Handle returns the replies to send, in order. A reply's Tx or Page tells
// the sender which row to link the sent message's ID to. Routes are tried in
// priority order; the first one that claims the message wins, and anything
// unclaimed is recorded as a transaction.
func (d *Dispatcher) Handle(ctx context.Context, in Incoming) []response.Reply {
	slog.Info("message received", "sender", in.SenderJID, "is_group", in.IsGroup, "group_id", in.GroupID, "text", in.Text, "stanza_id", in.StanzaID, "quoted_text", in.QuotedText)

	if isBotReply(in.Text) {
		return nil
	}
	if replies, ok := d.handleQuotedReply(ctx, in); ok {
		return replies
	}
	if replies, ok := d.handleCommand(ctx, in); ok {
		return replies
	}
	if replies, ok := d.handleReport(ctx, in); ok {
		return replies
	}
	return d.recordTransaction(ctx, in)
}

// handleQuotedReply resolves a reply against the bot message it quotes: a
// transaction confirmation (amend), a *PAGE n/N* summary (paging) or a quip.
// A stanza ID linked to none of them falls through to normal handling.
func (d *Dispatcher) handleQuotedReply(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	if in.StanzaID == "" {
		return nil, false
	}
	if replies, ok := d.amendTransaction(ctx, in); ok {
		return replies, true
	}
	if replies, ok := d.pageTransactions(ctx, in); ok {
		return replies, true
	}
	if strings.HasPrefix(in.QuotedText, message.QuipPrefix) {
		// A quip isn't linked to its transaction (the confirmation is),
		// so point the user there instead of treating this as new input.
		slog.Info("reply to a quip, pointing user to the confirmation", "sender", in.SenderJID, "stanza_id", in.StanzaID)
		return single(d.errorReply(message.PoolReplyToQuip, constant.ErrInvalidRequest.Code, in), nil), true
	}
	slog.Warn("reply stanza id not linked to a tracked transaction, falling back to normal handling",
		"sender", in.SenderJID, "stanza_id", in.StanzaID, "quoted_text", in.QuotedText)
	return nil, false
}

// handleCommand handles the plain-text commands: register and group split.
func (d *Dispatcher) handleCommand(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	switch {
	case strings.HasPrefix(in.Text, "register "):
		return d.register(ctx, in), true
	case in.Text == "split" && in.IsGroup:
		return d.split(ctx, in), true
	}
	return nil, false
}

// handleReport handles reporting requests: a summary of a period, or a
// paged listing of transactions.
func (d *Dispatcher) handleReport(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	if replies, ok := d.summarizeTransactions(ctx, in); ok {
		return replies, true
	}
	return d.queryTransactions(ctx, in)
}

// --- transactions ---

func (d *Dispatcher) recordTransaction(ctx context.Context, in Incoming) []response.Reply {
	replies, err := d.recordTx.Execute(ctx, in.SenderJID, in.Text, in.IsGroup, in.GroupID)
	if err != nil {
		util.LogError("record transaction failed", err, "sender", in.SenderJID)
		return single(d.replyForError(err, in), nil)
	}
	slog.Info("transaction processed", "sender", in.SenderJID, "is_group", in.IsGroup, "messages", len(replies))
	return replies
}

func (d *Dispatcher) amendTransaction(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	replies, handled, err := d.amendTx.Execute(ctx, in.SenderJID, in.StanzaID, in.Text)
	if err != nil {
		util.LogError("amend transaction failed", err, "sender", in.SenderJID)
		return single(d.replyForError(err, in), nil), true
	}
	if handled {
		slog.Info("transaction amended", "sender", in.SenderJID, "stanza_id", in.StanzaID, "messages", len(replies))
	}
	return replies, handled
}

// --- reports ---

func (d *Dispatcher) summarizeTransactions(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	summary, handled, err := d.summarizeTx.Execute(ctx, in.SenderJID, in.Text)
	if err != nil {
		util.LogError("summarize transactions failed", err, "sender", in.SenderJID)
		return single(d.replyForError(err, in), nil), true
	}
	if !handled {
		return nil, false
	}
	slog.Info("transactions summarized", "sender", in.SenderJID)
	return single(summary, nil), true
}

func (d *Dispatcher) queryTransactions(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	replies, handled, err := d.queryTx.Execute(ctx, in.SenderJID, in.Text)
	if err != nil {
		util.LogError("query transactions failed", err, "sender", in.SenderJID)
		return single(d.replyForError(err, in), nil), true
	}
	if handled {
		slog.Info("transactions listed", "sender", in.SenderJID, "messages", len(replies))
	}
	return replies, handled
}

func (d *Dispatcher) pageTransactions(ctx context.Context, in Incoming) ([]response.Reply, bool) {
	replies, handled, err := d.pageTx.Execute(ctx, in.SenderJID, in.StanzaID, in.Text)
	if err != nil {
		util.LogError("page transactions failed", err, "sender", in.SenderJID)
		return single(d.replyForError(err, in), nil), true
	}
	if handled {
		slog.Info("transactions paged", "sender", in.SenderJID, "stanza_id", in.StanzaID, "messages", len(replies))
	}
	return replies, handled
}

// --- commands ---

func (d *Dispatcher) register(ctx context.Context, in Incoming) []response.Reply {
	parts := strings.Fields(in.Text)
	if len(parts) < 3 {
		return single(message.RegisterUsage, nil)
	}
	reply, err := d.registerUser.Execute(ctx, in.SenderJID, parts[1], parts[2])
	if err != nil {
		util.LogError("register failed", err, "sender", in.SenderJID)
		return single(d.replyForError(err, in), nil)
	}
	slog.Info("user registered", "sender", in.SenderJID, "name", parts[1])
	return single(reply, nil)
}

func (d *Dispatcher) split(ctx context.Context, in Incoming) []response.Reply {
	reply, err := d.computeSplit.Execute(ctx, in.GroupID)
	if err != nil {
		util.LogError("split failed", err, "group_id", in.GroupID)
		return single(d.replyForError(err, in), nil)
	}
	slog.Info("split computed", "group_id", in.GroupID)
	return single(reply, nil)
}

// single wraps one usecase result as the reply list; tx may be nil.
func single(text string, tx *domain.Transaction) []response.Reply {
	if text == "" {
		return nil
	}
	return []response.Reply{{Text: text, Tx: tx}}
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
func (d *Dispatcher) replyForError(err error, in Incoming) string {
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
	if c, ok := util.CodeOf(err); ok {
		code = c.Code
	}
	return d.errorReply(key, code, in)
}

// errorReply renders a Frankie error reply from pool key, ending with the
// emoji for code.
func (d *Dispatcher) errorReply(key message.PoolKey, code int, in Incoming) string {
	lang := message.ReplyLang
	if lang == "" {
		lang = d.persona.Lang(in.SenderJID, in.Text)
	}
	msg := d.persona.Error(key, in.SenderJID, lang)

	emoji, ok := message.ErrorEmoji[code]
	if !ok {
		emoji = message.ErrorEmojiDefault
	}
	slog.Info("error reply sent", "sender", in.SenderJID, "code", code, "emoji", emoji, "pool", key)
	return fmt.Sprintf(message.ErrWithCode, msg, emoji)
}
