package usecase

import (
	"context"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/message"
	"whatsup-bot/internal/persona"
	"whatsup-bot/internal/port"
)

type fakeUserRepo struct{ user *domain.User }

func (f *fakeUserRepo) Save(context.Context, *domain.User) error { return nil }
func (f *fakeUserRepo) FindByJID(context.Context, string) (*domain.User, error) {
	return f.user, nil
}
func (f *fakeUserRepo) FindByID(context.Context, int64) (*domain.User, error) { return f.user, nil }

type fakeTxRepo struct {
	port.TransactionRepository // unused methods panic if called
	saved                      []*domain.Transaction
}

func (f *fakeTxRepo) Save(_ context.Context, tx *domain.Transaction) error {
	tx.ID = int64(len(f.saved) + 1)
	f.saved = append(f.saved, tx)
	return nil
}

type fakeParser struct {
	port.MessageParser
	parsed *port.ParsedMessage
}

func (f *fakeParser) Parse(context.Context, string) (*port.ParsedMessage, error) {
	return f.parsed, nil
}

func newRecordUC(parsed *port.ParsedMessage, picker *persona.Picker) *RecordTransactionUseCase {
	return NewRecordTransactionUseCase(
		&fakeUserRepo{user: &domain.User{ID: 1, Name: "Chris"}},
		&fakeTxRepo{},
		&fakeParser{parsed: parsed},
		picker,
	)
}

func TestRenderTransactionReplyWithoutQuipUnchanged(t *testing.T) {
	tx := &domain.Transaction{
		Type: domain.Outcome, Description: "nasi goreng", Category: domain.CategoryFood,
		Amount: 5000, TransactionDate: time.Date(2026, 10, 4, 0, 0, 0, 0, time.UTC),
	}
	want := "*EXPENSE*\n- Desc: *Nasi Goreng*\n- Category: *Food*\n- Amount: *IDR 5,000*\n- Date: *04 Oct 2026*\n\n_*Reply to this message to update or delete this transaction_"
	if got := renderTransactionReply(tx, ""); got != want {
		t.Errorf("reply without quip changed:\n got: %q\nwant: %q", got, want)
	}

	withQuip := renderTransactionReply(tx, "Frankie wants some too")
	wantQuip := "- Date: *04 Oct 2026*\n\nFrankie wants some too\n\n_*Reply"
	if !strings.Contains(withQuip, wantQuip) || !strings.HasPrefix(withQuip, "*EXPENSE*") {
		t.Errorf("quip not placed above footer:\n%s", withQuip)
	}
}

func TestRecordTransactionQuipRate(t *testing.T) {
	parsed := &port.ParsedMessage{
		Valid: true, Type: "DB", Amount: 5000, Category: "food",
		Description: "nasi goreng", Date: "2026-10-04", Quip: "Nasi goreng 5 ribu?! Frankie mau juga, bos!",
	}
	picker := persona.NewWithSource(rand.NewPCG(11, 11), time.Now)
	uc := newRecordUC(parsed, picker)

	const n = 2000
	withQuip := 0
	for i := 0; i < n; i++ {
		reply, tx, err := uc.Execute(context.Background(), "jid", "beli nasi goreng 5000", false, 0)
		if err != nil || tx == nil {
			t.Fatalf("Execute: %v", err)
		}
		if !strings.HasPrefix(reply, "*EXPENSE*") {
			t.Fatalf("reply must start with *EXPENSE* for the self-echo guard: %q", reply)
		}
		if strings.Contains(reply, parsed.Quip) {
			withQuip++
		}
	}
	if pct := withQuip * 100 / n; pct < message.QuipChancePct-4 || pct > message.QuipChancePct+4 {
		t.Errorf("quip shown %d%% of the time, want about %d%%", pct, message.QuipChancePct)
	}
}

func TestRecordTransactionFallbackQuip(t *testing.T) {
	parsed := &port.ParsedMessage{
		Valid: true, Type: "CR", Amount: 10000000, Category: "salary",
		Description: "gaji", Date: "2026-10-04", Quip: "  **  ",
	}
	uc := newRecordUC(parsed, persona.NewWithSource(rand.NewPCG(5, 5), time.Now))

	for i := 0; i < 200; i++ {
		reply, _, _ := uc.Execute(context.Background(), "jid", "gaji 10jt", false, 0)
		for _, q := range message.QuipIncome[message.LangID] {
			if strings.Contains(reply, q) {
				return // fallback income quip, in Indonesian
			}
		}
	}
	t.Error("empty model quip never fell back to the Indonesian income pool")
}

func TestRecordTransactionNoQuipOnAmountPrompt(t *testing.T) {
	parsed := &port.ParsedMessage{
		Valid: true, Type: "DB", Amount: 0, Category: "food",
		Description: "donut", Date: "2026-10-04", Quip: "Donut time!",
	}
	uc := newRecordUC(parsed, persona.NewWithSource(rand.NewPCG(1, 1), time.Now))
	for i := 0; i < 50; i++ {
		reply, _, _ := uc.Execute(context.Background(), "jid", "beli donut", false, 0)
		if strings.Contains(reply, "Donut time!") {
			t.Fatalf("amount prompt should not carry a quip: %q", reply)
		}
	}
}
