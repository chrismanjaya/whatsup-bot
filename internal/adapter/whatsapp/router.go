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
	computeSplit *usecase.ComputeSplitUseCase
}

func (r *router) handle(ctx context.Context, in incoming) (string, *domain.Transaction) {
	text := in.text
	slog.Info("message received", "sender", in.senderJID, "is_group", in.isGroup, "group_id", in.groupID, "text", text, "stanza_id", in.stanzaID, "quoted_text", in.quotedText)

	if isBotReply(text) {
		return "", nil
	}

	if in.stanzaID != "" {
		reply, tx, handled, err := r.amendTx.Execute(ctx, in.senderJID, in.stanzaID, text)
		if err != nil {
			utils.LogError("amend transaction failed", err, "sender", in.senderJID)
			return replyForError(err), nil
		}
		if handled {
			slog.Info("transaction amended", "sender", in.senderJID, "stanza_id", in.stanzaID)
			return reply, tx
		}
		slog.Warn("reply stanza id not linked to a tracked transaction, falling back to normal handling",
			"sender", in.senderJID, "stanza_id", in.stanzaID, "quoted_text", in.quotedText)
	}

	if strings.HasPrefix(text, "register ") {
		parts := strings.Fields(text)
		if len(parts) < 3 {
			return message.RegisterUsage, nil
		}
		reply, err := r.registerUser.Execute(ctx, in.senderJID, parts[1], parts[2])
		if err != nil {
			utils.LogError("register failed", err, "sender", in.senderJID)
			return replyForError(err), nil
		}
		slog.Info("user registered", "sender", in.senderJID, "name", parts[1])
		return reply, nil
	}

	if text == "split" && in.isGroup {
		reply, err := r.computeSplit.Execute(ctx, in.groupID)
		if err != nil {
			utils.LogError("split failed", err, "group_id", in.groupID)
			return replyForError(err), nil
		}
		slog.Info("split computed", "group_id", in.groupID)
		return reply, nil
	}

	reply, tx, err := r.recordTx.Execute(ctx, in.senderJID, text, in.isGroup, in.groupID)
	if err != nil {
		utils.LogError("record transaction failed", err, "sender", in.senderJID)
		return replyForError(err), nil
	}
	slog.Info("transaction processed", "sender", in.senderJID, "is_group", in.isGroup, "recorded", tx != nil)
	return reply, tx
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
// numeric error code is appended so a user can report it without exposing
// any internal detail — a developer can look up what it means from there.
func replyForError(err error) string {
	msg := message.ErrGeneric
	switch {
	case errors.Is(err, constant.ErrAlreadyExists):
		msg = message.ErrAlreadyRegistered
	case errors.Is(err, constant.ErrNotFound):
		msg = message.ErrNotRegistered
	case errors.Is(err, constant.ErrInvalidRequest):
		msg = message.ErrInvalidRequest
	case errors.Is(err, constant.ErrServiceUnavailable):
		msg = message.ErrServiceUnavailable
	}

	code := constant.ErrInternal.Code
	if c, ok := utils.CodeOf(err); ok {
		code = c.Code
	}
	return fmt.Sprintf(message.ErrWithCode, msg, code)
}
