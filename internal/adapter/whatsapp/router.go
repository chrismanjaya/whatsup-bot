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
// the handler which row to link the sent message's WhatsApp ID to.
func (r *router) handle(ctx context.Context, in incoming) []usecase.Reply {
	text := in.text
	slog.Info("message received", "sender", in.senderJID, "is_group", in.isGroup, "group_id", in.groupID, "text", text, "stanza_id", in.stanzaID, "quoted_text", in.quotedText)

	if isBotReply(text) {
		return nil
	}

	if in.stanzaID != "" {
		amendReplies, handled, err := r.amendTx.Execute(ctx, in.senderJID, in.stanzaID, text)
		if err != nil {
			utils.LogError("amend transaction failed", err, "sender", in.senderJID)
			return single(r.replyForError(err, in), nil)
		}
		if handled {
			slog.Info("transaction amended", "sender", in.senderJID, "stanza_id", in.stanzaID, "messages", len(amendReplies))
			return amendReplies
		}

		pageReplies, handled, err := r.pageTx.Execute(ctx, in.senderJID, in.stanzaID, text)
		if err != nil {
			utils.LogError("page transactions failed", err, "sender", in.senderJID)
			return single(r.replyForError(err, in), nil)
		}
		if handled {
			slog.Info("transactions paged", "sender", in.senderJID, "stanza_id", in.stanzaID, "messages", len(pageReplies))
			return pageReplies
		}

		if strings.HasPrefix(in.quotedText, message.QuipPrefix) {
			// A quip isn't linked to its transaction (the confirmation is),
			// so point the user there instead of treating this as new input.
			slog.Info("reply to a quip, pointing user to the confirmation", "sender", in.senderJID, "stanza_id", in.stanzaID)
			return single(r.errorReply(message.PoolReplyToQuip, constant.ErrInvalidRequest.Code, in), nil)
		}

		slog.Warn("reply stanza id not linked to a tracked transaction, falling back to normal handling",
			"sender", in.senderJID, "stanza_id", in.stanzaID, "quoted_text", in.quotedText)
	}

	if strings.HasPrefix(text, "register ") {
		parts := strings.Fields(text)
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

	if text == "split" && in.isGroup {
		reply, err := r.computeSplit.Execute(ctx, in.groupID)
		if err != nil {
			utils.LogError("split failed", err, "group_id", in.groupID)
			return single(r.replyForError(err, in), nil)
		}
		slog.Info("split computed", "group_id", in.groupID)
		return single(reply, nil)
	}

	summary, handled, err := r.summarizeTx.Execute(ctx, in.senderJID, text)
	if err != nil {
		utils.LogError("summarize transactions failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil)
	}
	if handled {
		slog.Info("transactions summarized", "sender", in.senderJID)
		return single(summary, nil)
	}

	queryReplies, handled, err := r.queryTx.Execute(ctx, in.senderJID, text)
	if err != nil {
		utils.LogError("query transactions failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil)
	}
	if handled {
		slog.Info("transactions listed", "sender", in.senderJID, "messages", len(queryReplies))
		return queryReplies
	}

	recordReplies, err := r.recordTx.Execute(ctx, in.senderJID, text, in.isGroup, in.groupID)
	if err != nil {
		utils.LogError("record transaction failed", err, "sender", in.senderJID)
		return single(r.replyForError(err, in), nil)
	}
	slog.Info("transaction processed", "sender", in.senderJID, "is_group", in.isGroup, "messages", len(recordReplies))
	return recordReplies
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
