package whatsapp

import (
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func newEvent(chat types.JID, message *waE2E.Message) *events.Message {
	return &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat:   chat,
				Sender: types.NewJID("628111", types.DefaultUserServer),
			},
		},
		Message: message,
	}
}

func TestParseIncomingMessage(t *testing.T) {
	dm := types.NewJID("628111", types.DefaultUserServer)
	group := types.NewJID("120363", types.GroupServer)

	t.Run("returns nil for non-text messages", func(t *testing.T) {
		sticker := &waE2E.Message{StickerMessage: &waE2E.StickerMessage{}}
		if got := parseIncomingMessage(newEvent(dm, sticker)); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
	})

	t.Run("returns nil for whitespace-only text", func(t *testing.T) {
		blank := &waE2E.Message{Conversation: proto.String("   ")}
		if got := parseIncomingMessage(newEvent(dm, blank)); got != nil {
			t.Fatalf("got %+v, want nil", got)
		}
	})

	t.Run("parses plain text", func(t *testing.T) {
		text := &waE2E.Message{Conversation: proto.String(" beli kopi 25rb ")}
		got := parseIncomingMessage(newEvent(dm, text))
		if got == nil || got.Text != "beli kopi 25rb" || got.IsGroup || got.IsReply {
			t.Fatalf("got %+v, want trimmed direct non-reply text", got)
		}
	})

	t.Run("parses a reply with quoted text in a group", func(t *testing.T) {
		reply := &waE2E.Message{ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: proto.String("hapus"),
			ContextInfo: &waE2E.ContextInfo{
				StanzaID:      proto.String("ABC123"),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("*EXPENSE*")},
			},
		}}
		got := parseIncomingMessage(newEvent(group, reply))
		if got == nil || !got.IsReply || !got.IsGroup || got.StanzaID != "ABC123" || got.QuotedText != "*EXPENSE*" {
			t.Fatalf("got %+v, want group reply with stanza and quoted text", got)
		}
	})
}
