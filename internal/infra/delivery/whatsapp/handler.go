package whatsapp

import (
	"context"
	"strings"
	"time"

	whatsapp_client "whatsupbot/internal/infra/external/whatsapp"
	"whatsupbot/internal/infra/service"
	"whatsupbot/internal/usecase/render"
	"whatsupbot/internal/util"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const (
	// The pause between consecutive data messages of one response (a Reply with Typing set gets a typing pause instead).
	sendInterval = 250 * time.Millisecond

	// How long a chat's worker waits for another message before it exits (see chatQueue).
	chatIdle = 5 * time.Minute

	typingBase    = 700 * time.Millisecond
	typingPerRune = 20 * time.Millisecond
	typingMax     = 2 * time.Second
)

type Handler struct {
	whatsapp       *whatsapp_client.Client
	queue          *chatQueue
	chatbotService *service.Chatbot
}

func NewHandler(
	whatsapp *whatsapp_client.Client,
	chatbotService *service.Chatbot,
) *Handler {
	return &Handler{
		whatsapp:       whatsapp,
		queue:          newChatQueue(chatIdle),
		chatbotService: chatbotService,
	}
}

func (h *Handler) Client() *whatsapp_client.Client {
	return h.whatsapp
}

func (h *Handler) Register() {
	h.whatsapp.Client().AddEventHandler(h.handleEvent)
}

func (h *Handler) Connect(ctx context.Context) error {
	return h.whatsapp.Connect(ctx)
}

func (h *Handler) Disconnect() {
	h.whatsapp.Disconnect()
}

func (h *Handler) handleEvent(event interface{}) {
	eventMessage, ok := event.(*events.Message)
	if !ok {
		return
	}

	if eventMessage.Info.IsFromMe {
		// ignore self chat
		return
	}

	// parse to incoming message; ignore non-text events (stickers, media, reactions)
	message := parseIncomingMessage(eventMessage)
	if message == nil {
		return
	}

	// add to queue
	chatJID := eventMessage.Info.Chat
	h.queue.enqueue(chatJID.String(), func() {
		h.process(context.Background(), chatJID, message)
	})
}

func (h *Handler) process(ctx context.Context, chatJID types.JID, message *service.IncomingMessage) {
	startTyping := time.Now()
	h.composing(ctx, chatJID, true)

	// chatbot service
	replies := h.chatbotService.ProcessIncomingMessage(ctx, message)
	if len(replies) == 0 {
		h.composing(ctx, chatJID, false)
	}

	// extract replay
	for _, reply := range replies {
		h.typing(ctx, chatJID, reply.Text, startTyping)
		startTyping = time.Now()
		h.send(ctx, chatJID, reply)

		time.Sleep(sendInterval)
	}
}

func (h *Handler) composing(ctx context.Context, chatJID types.JID, isProcessing bool) {
	var state types.ChatPresence
	if isProcessing {
		state = types.ChatPresenceComposing
	} else {
		state = types.ChatPresencePaused
	}

	if err := h.whatsapp.Client().SendChatPresence(ctx, chatJID, state, types.ChatPresenceMediaText); err != nil {
		util.LogError("send typing presence failed", err, "chat_jid", chatJID.String())
	}
}

func (h *Handler) typing(ctx context.Context, chatJID types.JID, text string, typingSince time.Time) {
	h.composing(ctx, chatJID, true)
	time.Sleep(typingDuration(text, typingSince))
}

func (h *Handler) send(ctx context.Context, chatJID types.JID, reply render.Reply) {
	msg := &waE2E.Message{Conversation: proto.String(reply.Text)}
	resp, err := h.whatsapp.Client().SendMessage(ctx, chatJID, msg)
	if err != nil {
		util.LogError("send reply failed", err, "chat_jid", chatJID.String())
		return
	}

	// update transaction message ID
	if reply.Metadata.Transaction != nil {
		h.chatbotService.UpdateMessageID(ctx, reply.Metadata.Transaction.ID, resp.ID)
	}
}

func parseIncomingMessage(eventMessage *events.Message) *service.IncomingMessage {
	var (
		chatJID   = eventMessage.Info.Chat
		senderJID = eventMessage.Info.Sender.ToNonAD().String()

		text = eventMessage.Message.GetConversation()

		extended   = eventMessage.Message.GetExtendedTextMessage()
		stanzaID   = ""
		quotedText = ""
		groupName  = ""

		isGroup bool
		isReply bool
	)

	// reply message
	if extended != nil {
		if text == "" {
			text = extended.GetText()
		}
		if ctxInfo := extended.GetContextInfo(); ctxInfo != nil {
			stanzaID = ctxInfo.GetStanzaID()
			if quotedMessage := ctxInfo.GetQuotedMessage(); quotedMessage != nil {
				quotedText = quotedMessage.GetConversation()
				if quotedText == "" {
					quotedText = quotedMessage.GetExtendedTextMessage().GetText()
				}
			}
		}
	}

	text = strings.TrimSpace(text)
	if text == "" {
		// invalid message
		return nil
	}

	isGroup = strings.HasSuffix(chatJID.String(), "@g.us")
	if isGroup {
		groupName = chatJID.String()
	}

	if stanzaID != "" {
		isReply = true
	}

	return &service.IncomingMessage{
		SenderJID:  senderJID,
		Text:       text,
		StanzaID:   stanzaID,
		QuotedText: quotedText,
		IsReply:    isReply,
		IsGroup:    isGroup,
		GroupName:  groupName,
	}
}
