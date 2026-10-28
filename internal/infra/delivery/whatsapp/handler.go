package whatsapp

import (
	"context"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
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

	// record batcher goroutine, started by Connect and drained by Disconnect
	batcherWG sync.WaitGroup
}

// replyTarget is the incoming message a reaction or quoted reply points at.
type replyTarget struct {
	chat      types.JID
	sender    types.JID
	messageID string
	text      string
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

// Connect connects to WhatsApp and starts the record batcher, which runs until ctx is cancelled.
func (h *Handler) Connect(ctx context.Context) error {
	if err := h.whatsapp.Connect(ctx); err != nil {
		return err
	}

	h.batcherWG.Add(1)
	go func() {
		defer h.batcherWG.Done()
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic", "where", "record batcher", "panic", r, "stack", string(debug.Stack()))
			}
		}()
		h.chatbotService.RunRecordBatcher(ctx, h.deliverRecord)
	}()

	return nil
}

// Disconnect waits for the record batcher to stop (cancel Connect's ctx first), then disconnects.
func (h *Handler) Disconnect() {
	h.batcherWG.Wait()
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
	target := replyTarget{
		chat:      chatJID,
		sender:    eventMessage.Info.Sender.ToNonAD(),
		messageID: eventMessage.Info.ID,
		text:      message.Text,
	}
	h.queue.enqueue(chatJID.String(), func() {
		h.process(context.Background(), chatJID, target, message)
	})
}

func (h *Handler) process(ctx context.Context, chatJID types.JID, target replyTarget, message *service.IncomingMessage) {
	h.composing(ctx, chatJID, true)

	// chatbot service
	replies := h.chatbotService.ProcessIncomingMessage(ctx, message)

	h.sendReplies(ctx, target, replies, true)
}

// deliverRecord answers a record message once its batch is parsed, through the chat's queue so it stays in order with that chat's other replies. It skips the typing pause: the answer is already late, and pausing per line would hold up instant replies (amend, reports) queued behind it in the same chat.
func (h *Handler) deliverRecord(ctx context.Context, delivery service.RecordDelivery) {
	chatJID, err := types.ParseJID(delivery.Message.ChatJID)
	if err != nil {
		util.LogError("parse chat jid failed", err, "chat_jid", delivery.Message.ChatJID)
		return
	}

	senderJID, err := types.ParseJID(delivery.Message.SenderJID)
	if err != nil {
		util.LogError("parse sender jid failed", err, "chat_jid", delivery.Message.ChatJID)
		return
	}

	target := replyTarget{
		chat:      chatJID,
		sender:    senderJID,
		messageID: delivery.Message.MessageID,
		text:      delivery.Message.Text,
	}

	h.queue.enqueue(chatJID.String(), func() {
		h.sendReplies(ctx, target, delivery.Replies, false)
	})
}

// sendReplies sends replies in order; withTyping shows a typing pause before each text reply.
func (h *Handler) sendReplies(ctx context.Context, target replyTarget, replies []render.Reply, withTyping bool) {
	var (
		startTyping = time.Now()
		hasText     bool
	)

	for _, reply := range replies {
		if reply.Reaction != "" {
			h.react(ctx, target, reply.Reaction)
			continue
		}

		hasText = true
		if withTyping {
			h.typing(ctx, target.chat, reply.Text, startTyping)
			startTyping = time.Now()
		}
		h.send(ctx, target, reply)

		time.Sleep(sendInterval)
	}

	if withTyping && !hasText {
		h.composing(ctx, target.chat, false)
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

func (h *Handler) send(ctx context.Context, target replyTarget, reply render.Reply) {
	msg := &waE2E.Message{Conversation: proto.String(reply.Text)}
	if reply.IsQuoted {
		msg = &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String(reply.Text),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String(target.messageID),
				Participant:   proto.String(target.sender.String()),
				QuotedMessage: &waE2E.Message{Conversation: proto.String(target.text)},
			},
		}}
	}

	resp, err := h.whatsapp.Client().SendMessage(ctx, target.chat, msg)
	if err != nil {
		util.LogError("send reply failed", err, "chat_jid", target.chat.String())
		return
	}

	// update transaction message ID
	if reply.Metadata.Transaction != nil {
		if err := h.chatbotService.UpdateMessageID(ctx, reply.Metadata.Transaction.ID, resp.ID); err != nil {
			util.LogError("update transaction message id failed", err, "chat_jid", target.chat.String())
		}
	}
}

// react sets the bot's reaction on the target message; a new reaction replaces the previous one.
func (h *Handler) react(ctx context.Context, target replyTarget, reaction string) {
	msg := h.whatsapp.Client().BuildReaction(target.chat, target.sender, target.messageID, reaction)
	if _, err := h.whatsapp.Client().SendMessage(ctx, target.chat, msg); err != nil {
		util.LogError("send reaction failed", err, "chat_jid", target.chat.String())
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
		MessageID:  eventMessage.Info.ID,
		ChatJID:    chatJID.String(),
		SenderJID:  senderJID,
		ReceivedAt: eventMessage.Info.Timestamp.In(util.Jakarta),
		Text:       text,
		StanzaID:   stanzaID,
		QuotedText: quotedText,
		IsReply:    isReply,
		IsGroup:    isGroup,
		GroupName:  groupName,
	}
}
