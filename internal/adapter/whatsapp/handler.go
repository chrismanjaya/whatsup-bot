package whatsapp

import (
	"context"
	"strings"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"whatsup-bot/internal/port"
	"whatsup-bot/internal/usecase"
	"whatsup-bot/internal/utils"
)

// Handler is the whatsmeow event handler: it translates WhatsApp events into
// plain inputs, delegates to the router, and sends the reply back.
type Handler struct {
	client           *whatsmeow.Client
	txRepo           port.TransactionRepository
	ensureGroup      *usecase.EnsureGroupUseCase
	ensureMembership *usecase.EnsureGroupMembershipUseCase
	router           router
}

func NewHandler(
	client *whatsmeow.Client,
	txRepo port.TransactionRepository,
	registerUser *usecase.RegisterUserUseCase,
	recordTx *usecase.RecordTransactionUseCase,
	amendTx *usecase.AmendTransactionUseCase,
	ensureGroup *usecase.EnsureGroupUseCase,
	ensureMembership *usecase.EnsureGroupMembershipUseCase,
	computeSplit *usecase.ComputeSplitUseCase,
) *Handler {
	return &Handler{
		client:           client,
		txRepo:           txRepo,
		ensureGroup:      ensureGroup,
		ensureMembership: ensureMembership,
		router: router{
			registerUser: registerUser,
			recordTx:     recordTx,
			amendTx:      amendTx,
			computeSplit: computeSplit,
		},
	}
}

// Register subscribes the handler to the client's events.
func (h *Handler) Register() {
	h.client.AddEventHandler(h.handleEvent)
}

func (h *Handler) handleEvent(evt interface{}) {
	ctx := context.Background()

	v, ok := evt.(*events.Message)
	if !ok {
		return
	}
	if v.Info.IsFromMe {
		return
	}

	text := v.Message.GetConversation()
	stanzaID := ""
	quotedText := ""
	if ext := v.Message.GetExtendedTextMessage(); ext != nil {
		if text == "" {
			text = ext.GetText()
		}
		if ci := ext.GetContextInfo(); ci != nil {
			stanzaID = ci.GetStanzaID()
			if qm := ci.GetQuotedMessage(); qm != nil {
				quotedText = qm.GetConversation()
				if quotedText == "" {
					quotedText = qm.GetExtendedTextMessage().GetText()
				}
			}
		}
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	senderJID := v.Info.Sender.ToNonAD().String()
	chatJID := v.Info.Chat
	isGroup := strings.HasSuffix(chatJID.String(), "@g.us")

	var groupID int64
	if isGroup {
		groupName := chatJID.String()
		if info, err := h.client.GetGroupInfo(ctx, chatJID); err == nil {
			groupName = info.Name
		}
		gid, err := h.ensureGroup.Execute(ctx, chatJID.String(), groupName)
		if err != nil {
			utils.LogError("ensure group failed", err, "group_jid", chatJID.String())
			return
		}
		groupID = gid

		if err := h.ensureMembership.Execute(ctx, senderJID, groupID); err != nil {
			utils.LogError("ensure membership failed", err, "sender", senderJID, "group_id", groupID)
		}
	}

	if err := h.client.SendChatPresence(ctx, chatJID, types.ChatPresenceComposing, types.ChatPresenceMediaText); err != nil {
		utils.LogError("send typing presence failed", err, "chat_jid", chatJID.String())
	}

	reply, tx := h.router.handle(ctx, incoming{
		text:       text,
		stanzaID:   stanzaID,
		quotedText: quotedText,
		senderJID:  senderJID,
		isGroup:    isGroup,
		groupID:    groupID,
	})

	if err := h.client.SendChatPresence(ctx, chatJID, types.ChatPresencePaused, types.ChatPresenceMediaText); err != nil {
		utils.LogError("clear typing presence failed", err, "chat_jid", chatJID.String())
	}

	if reply == "" {
		return
	}

	msg := &waE2E.Message{Conversation: proto.String(reply)}
	resp, err := h.client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		utils.LogError("send reply failed", err, "chat_jid", chatJID.String())
		return
	}

	if tx != nil {
		if err := h.txRepo.SetWAMessageID(ctx, tx.ID, resp.ID); err != nil {
			utils.LogError("set wa message id failed", err, "tx_id", tx.ID)
		}
	}
}
