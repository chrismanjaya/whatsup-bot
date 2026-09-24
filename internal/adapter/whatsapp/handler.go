package whatsapp

import (
	"context"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"

	"whatsup-bot/internal/port"
	"whatsup-bot/internal/usecase"
	"whatsup-bot/internal/utils"
)

// sendInterval is the pause between consecutive messages of one response.
const sendInterval = 250 * time.Millisecond

// Handler is the whatsmeow event handler: it translates WhatsApp events into
// plain inputs, delegates to the router, and sends the reply back.
type Handler struct {
	client           *whatsmeow.Client
	txRepo           port.TransactionRepository
	pageRepo         port.QueryPageRepository
	ensureGroup      *usecase.EnsureGroupUseCase
	ensureMembership *usecase.EnsureGroupMembershipUseCase
	router           router
}

func NewHandler(
	client *whatsmeow.Client,
	txRepo port.TransactionRepository,
	pageRepo port.QueryPageRepository,
	registerUser *usecase.RegisterUserUseCase,
	recordTx *usecase.RecordTransactionUseCase,
	amendTx *usecase.AmendTransactionUseCase,
	queryTx *usecase.QueryTransactionsUseCase,
	pageTx *usecase.PageTransactionsUseCase,
	ensureGroup *usecase.EnsureGroupUseCase,
	ensureMembership *usecase.EnsureGroupMembershipUseCase,
	computeSplit *usecase.ComputeSplitUseCase,
) *Handler {
	return &Handler{
		client:           client,
		txRepo:           txRepo,
		pageRepo:         pageRepo,
		ensureGroup:      ensureGroup,
		ensureMembership: ensureMembership,
		router: router{
			registerUser: registerUser,
			recordTx:     recordTx,
			amendTx:      amendTx,
			queryTx:      queryTx,
			pageTx:       pageTx,
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

	replies := h.router.handle(ctx, incoming{
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

	for i, reply := range replies {
		if i > 0 {
			// Keep a multi-message listing in order on the recipient's phone.
			time.Sleep(sendInterval)
		}
		h.send(ctx, chatJID, reply)
	}
}

// send delivers one reply and links the sent message's WhatsApp ID to the
// transaction or query page it represents, so replying to it can be resolved.
func (h *Handler) send(ctx context.Context, chatJID types.JID, reply usecase.Reply) {
	msg := &waE2E.Message{Conversation: proto.String(reply.Text)}
	resp, err := h.client.SendMessage(ctx, chatJID, msg)
	if err != nil {
		utils.LogError("send reply failed", err, "chat_jid", chatJID.String())
		return
	}

	if reply.Tx != nil {
		if err := h.txRepo.SetWAMessageID(ctx, reply.Tx.ID, resp.ID); err != nil {
			utils.LogError("set wa message id failed", err, "tx_id", reply.Tx.ID)
		}
	}
	if reply.Page != nil {
		if err := h.pageRepo.SetWAMessageID(ctx, reply.Page.ID, resp.ID); err != nil {
			utils.LogError("set query page wa message id failed", err, "page_id", reply.Page.ID)
		}
	}
}
