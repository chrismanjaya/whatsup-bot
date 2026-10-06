package usecase

import (
	"context"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"whatsup-bot/internal/domain"
	"whatsup-bot/internal/parser"
	"whatsup-bot/internal/repository"
	"whatsup-bot/internal/usecase/group"
	"whatsup-bot/internal/usecase/message"
	"whatsup-bot/internal/usecase/persona"
	"whatsup-bot/internal/usecase/transaction"
	"whatsup-bot/internal/usecase/user"
)

// The fakes embed the interface, so a route that reaches an unexpected
// method panics instead of passing silently.

type fakeUserRepo struct {
	repository.UserRepository
	user *domain.User
}

func (f *fakeUserRepo) FindByJID(context.Context, string) (*domain.User, error) { return f.user, nil }

type fakeTxRepo struct {
	repository.TransactionRepository
}

func (fakeTxRepo) FindByWAMessageID(context.Context, string) (*domain.Transaction, error) {
	return nil, nil
}

type fakePageRepo struct{ repository.QueryPageRepository }

func (fakePageRepo) FindByWAMessageID(context.Context, string) (*domain.QueryPage, error) {
	return nil, nil
}

type fakeParser struct{ parser.MessageParser }

// newTestDispatcher wires the real usecases to fakes where no stanza ID is
// linked to a transaction or page, and the sender is u (nil = unregistered).
func newTestDispatcher(u *domain.User) *Dispatcher {
	users := &fakeUserRepo{user: u}
	txs, pages, p := fakeTxRepo{}, fakePageRepo{}, fakeParser{}
	picker := persona.NewWithSource(rand.NewPCG(1, 1), time.Now)
	return NewDispatcher(
		user.NewRegisterUseCase(users),
		transaction.NewRecordUseCase(users, txs, p, picker),
		transaction.NewAmendUseCase(users, txs, p, picker),
		transaction.NewSummarizeUseCase(users, txs, p),
		transaction.NewQueryUseCase(users, txs, pages, p),
		transaction.NewPageUseCase(users, txs, pages),
		group.NewComputeSplitUseCase(nil, txs),
		picker,
	)
}

func TestDispatcherIgnoresBotEcho(t *testing.T) {
	d := newTestDispatcher(nil)
	if got := d.Handle(context.Background(), Incoming{Text: "*EXPENSE*\n- Desc: *Kopi*", SenderJID: "jid"}); got != nil {
		t.Errorf("bot's own reply should be ignored, got %+v", got)
	}
}

func TestDispatcherReplyToQuip(t *testing.T) {
	d := newTestDispatcher(&domain.User{ID: 1})
	got := d.Handle(context.Background(), Incoming{
		Text: "hapus", StanzaID: "wa-quip", QuotedText: message.QuipPrefix + "Coffee again?", SenderJID: "jid",
	})
	if len(got) != 1 || !strings.HasSuffix(got[0].Text, message.ErrorEmoji[400]) || got[0].Tx != nil {
		t.Errorf("reply to a quip should get one 400 error reply, got %+v", got)
	}
}

func TestDispatcherRegisterUsage(t *testing.T) {
	d := newTestDispatcher(nil)
	got := d.Handle(context.Background(), Incoming{Text: "register chris", SenderJID: "jid"})
	if len(got) != 1 || got[0].Text != message.RegisterUsage {
		t.Errorf("incomplete register should show usage, got %+v", got)
	}
}

func TestDispatcherUnregisteredSender(t *testing.T) {
	d := newTestDispatcher(nil)
	got := d.Handle(context.Background(), Incoming{Text: "kopi 20rb", SenderJID: "jid"})
	if len(got) != 1 || !strings.Contains(got[0].Text, "register <name> <email>") || !strings.HasSuffix(got[0].Text, message.ErrorEmoji[404]) {
		t.Errorf("unregistered sender should get the 404 register instruction, got %+v", got)
	}
}
