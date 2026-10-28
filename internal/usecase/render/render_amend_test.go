package render

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

func TestMain(m *testing.M) {
	util.InitTemplate()
	os.Exit(m.Run())
}

func TestAmendResponseSuccess(t *testing.T) {
	transaction := &domain.Transaction{
		ID:              7,
		Description:     "kopi",
		Type:            domain.Expense,
		Amount:          25000,
		Category:        domain.CategoryFood,
		TransactionDate: time.Date(2026, 10, 28, 0, 0, 0, 0, util.Jakarta),
	}

	t.Run("delete replies with the canned line quoted to the confirmation", func(t *testing.T) {
		replies := NewAmendResponseSuccess(context.Background(), transaction, false, true, "🗑️ Poof!").RenderSuccess()
		if len(replies) != 1 {
			t.Fatalf("got %d replies, want 1", len(replies))
		}

		reply := replies[0]
		if reply.Text != "🗑️ Poof!" || !reply.IsQuotedReplied || reply.IsQuoted {
			t.Fatalf("got %+v, want canned line quoted to the replied message", reply)
		}
		if reply.Metadata.Transaction != nil {
			t.Fatalf("deleted transaction must not get a new wa_message_id")
		}
	})

	t.Run("update replies with the new confirmation", func(t *testing.T) {
		replies := NewAmendResponseSuccess(context.Background(), transaction, true, false, "").RenderSuccess()
		if len(replies) != 1 || !strings.Contains(replies[0].Text, "*EXPENSE*") || replies[0].Metadata.Transaction != transaction {
			t.Fatalf("got %+v, want editable transaction confirmation", replies)
		}
	})
}
