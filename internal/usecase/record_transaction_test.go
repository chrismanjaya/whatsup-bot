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
		Description: "nasi goreng", Date: "2026-10-04", Quip: "Nasi goreng for 5 ribu?! Where is this place, bos?",
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
	if pct := withQuip * 100 / n; pct < message.QuipExpenseChancePct-4 || pct > message.QuipExpenseChancePct+4 {
		t.Errorf("quip shown %d%% of the time, want about %d%%", pct, message.QuipExpenseChancePct)
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
		for _, q := range message.QuipIncome {
			if strings.Contains(reply, q) {
				return // fallback income quip
			}
		}
	}
	t.Error("empty model quip never fell back to the income pool")
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

func TestRecordTransactionDropsIndonesianQuip(t *testing.T) {
	parsed := &port.ParsedMessage{
		Valid: true, Type: "CR", Amount: 509589, Category: "investment",
		Description: "bunga deposito", Date: "2026-09-15", Quip: "Wah, bunganya cair! Selamat ya, bos",
	}
	uc := newRecordUC(parsed, persona.NewWithSource(rand.NewPCG(8, 8), time.Now))
	reply, _, _ := uc.Execute(context.Background(), "jid", "15/9 deposito 509.589", false, 0)
	if strings.Contains(reply, parsed.Quip) {
		t.Fatalf("Indonesian model quip should be dropped:\n%s", reply)
	}
	found := false
	for _, q := range message.QuipIncome {
		if strings.Contains(reply, q) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected an English income pool quip instead:\n%s", reply)
	}
}

func TestRecordTransactionIncomeAlwaysQuips(t *testing.T) {
	parsed := &port.ParsedMessage{
		Valid: true, Type: "CR", Amount: 509589, Category: "investment",
		Description: "bunga deposito", Date: "2026-09-15", Quip: "Interest day! Your money works hard, bos 💰",
	}
	uc := newRecordUC(parsed, persona.NewWithSource(rand.NewPCG(2, 2), time.Now))
	for i := 0; i < 100; i++ {
		reply, _, _ := uc.Execute(context.Background(), "jid", "15/9 deposito 509.589", false, 0)
		if !strings.Contains(reply, parsed.Quip) {
			t.Fatalf("income without a quip at try %d:\n%s", i, reply)
		}
	}
}

type amendTxRepo struct {
	port.TransactionRepository
	tx *domain.Transaction
}

func (f *amendTxRepo) FindByWAMessageID(context.Context, string) (*domain.Transaction, error) {
	cp := *f.tx
	return &cp, nil
}
func (f *amendTxRepo) Update(context.Context, *domain.Transaction) error { return nil }

type amendParser struct {
	port.MessageParser
	result *port.AmendResult
}

func (f *amendParser) ParseAmend(context.Context, string, *port.CurrentTransaction) (*port.AmendResult, error) {
	return f.result, nil
}

func TestAmendFillingAmountQuips(t *testing.T) {
	pending := &domain.Transaction{
		ID: 7, UserID: 1, Type: domain.Income, Category: domain.CategoryInvestment,
		Description: "deposito", Amount: 0, TransactionDate: time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC),
	}
	result := &port.AmendResult{Action: "update", Type: "CR", Amount: 509589, Category: "investment", Description: "deposito", Date: "2026-09-04"}
	picker := persona.NewWithSource(rand.NewPCG(4, 4), time.Now)

	uc := NewAmendTransactionUseCase(&fakeUserRepo{user: &domain.User{ID: 1}}, &amendTxRepo{tx: pending}, &amendParser{result: result}, picker)
	reply, _, handled, err := uc.Execute(context.Background(), "jid", "wa-1", "509589")
	if err != nil || !handled {
		t.Fatalf("Execute: handled=%v err=%v", handled, err)
	}
	found := false
	for _, q := range message.QuipIncome {
		if strings.Contains(reply, q) {
			found = true
		}
	}
	if !found {
		t.Errorf("filling the amount of an income should add an income quip:\n%s", reply)
	}

	// A correction of an existing amount is not new: no quip.
	pending.Amount = 100000
	reply, _, _, _ = uc.Execute(context.Background(), "jid", "wa-1", "jadi 509589")
	if !strings.HasSuffix(reply, "_*Reply to this message to update or delete this transaction_") ||
		strings.Contains(reply, "\n\n\n") || strings.Count(reply, "\n\n") != 1 {
		t.Errorf("correction should render without a quip:\n%s", reply)
	}
}
