package transaction

import (
	"context"
	"errors"
	"testing"
	"time"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/util"
)

type fakeBatchParser struct {
	GeminiClient // unused methods panic if called

	received []domain.ParseBatchItem
	respond  func(items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error)
}

func (f *fakeBatchParser) ParseRecordBatch(ctx context.Context, items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
	f.received = items
	return f.respond(items)
}

type fakeTransactionRepo struct {
	TransactionRepository // unused methods panic if called

	created []*domain.Transaction
	failOn  string // Create fails for this description
}

func (f *fakeTransactionRepo) Create(ctx context.Context, transaction *domain.Transaction) error {
	if transaction.Description == f.failOn {
		return errors.New("disk full")
	}
	transaction.ID = int64(len(f.created) + 1)
	f.created = append(f.created, transaction)
	return nil
}

var receivedAt = time.Date(2026, 10, 28, 9, 0, 0, 0, util.Jakarta)

// validParsed echoes a valid expense for index, described by desc.
func validParsed(index int, desc string) *domain.ParsedMessage {
	return &domain.ParsedMessage{
		Index: index,
		Valid: true,
		Quip:  "quip " + desc,
		TransactionMessage: &domain.ParsedTransactionMessage{
			Type:        "DB",
			Amount:      25000,
			Category:    "food",
			Description: desc,
			Date:        "2026-10-28",
			DateLayout:  "2006-01-02",
		},
	}
}

func echoAll(items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
	messages := make([]*domain.ParsedMessage, 0, len(items))
	for _, item := range items {
		messages = append(messages, validParsed(item.Index, item.Text))
	}
	return messages, nil
}

func newTestRecordBatch(parser *fakeBatchParser, repo *fakeTransactionRepo) *RecordBatch {
	uc := NewRecordBatch(repo, parser)
	uc.pickIndex = func(n int) int { return n - 1 } // deterministic: last item of each user
	return uc
}

func inputsFor(userIDs ...int64) []RecordBatchInput {
	inputs := make([]RecordBatchInput, 0, len(userIDs))
	for i, userID := range userIDs {
		inputs = append(inputs, RecordBatchInput{
			UserID:     userID,
			Text:       string(rune('a' + i)),
			ReceivedAt: receivedAt,
		})
	}
	return inputs
}

func TestRecordBatchMapsIndexesBackToUsers(t *testing.T) {
	// parser answers out of order; results must still line up with inputs
	parser := &fakeBatchParser{respond: func(items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
		return []*domain.ParsedMessage{validParsed(3, "c"), validParsed(1, "a"), validParsed(2, "b")}, nil
	}}
	repo := &fakeTransactionRepo{}

	results, err := newTestRecordBatch(parser, repo).Execute(context.Background(), inputsFor(10, 20, 30))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, wantUser := range []int64{10, 20, 30} {
		got := results[i].Transaction
		if got == nil || got.UserID != wantUser || got.Description != string(rune('a'+i)) {
			t.Fatalf("result %d = %+v, want user %d", i, got, wantUser)
		}
	}

	for i, item := range parser.received {
		if item.Index != i+1 {
			t.Fatalf("item %d sent with index %d, want %d", i, item.Index, i+1)
		}
	}
}

func TestRecordBatchMissingAndDuplicateIndexes(t *testing.T) {
	parser := &fakeBatchParser{respond: func(items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
		// 1 ok, 2 duplicated, 3 missing, 99 out of range
		return []*domain.ParsedMessage{
			validParsed(1, "a"),
			validParsed(2, "b"),
			validParsed(2, "b-again"),
			validParsed(99, "ghost"),
		}, nil
	}}
	repo := &fakeTransactionRepo{}

	results, err := newTestRecordBatch(parser, repo).Execute(context.Background(), inputsFor(1, 1, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if results[0].Err != nil || results[0].Transaction == nil {
		t.Fatalf("index 1 should succeed, got %+v", results[0])
	}
	for _, i := range []int{1, 2} {
		if !errors.Is(results[i].Err, constant.ErrTransactionUnparsed) {
			t.Fatalf("result %d err = %v, want ErrTransactionUnparsed", i, results[i].Err)
		}
	}
	if len(repo.created) != 1 {
		t.Fatalf("created %d transactions, want 1", len(repo.created))
	}
}

func TestRecordBatchPartialFailure(t *testing.T) {
	parser := &fakeBatchParser{respond: func(items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
		notTransaction := validParsed(2, "hello")
		notTransaction.Valid = false

		badType := validParsed(3, "bad type")
		badType.TransactionMessage.Type = "XX"

		hugeAmount := validParsed(4, "huge")
		hugeAmount.TransactionMessage.Amount = maxTransactionAmount + 1

		future := validParsed(5, "future")
		future.TransactionMessage.Date = "2026-10-29"

		yesterday := validParsed(7, "yesterday")
		yesterday.TransactionMessage.Date = "2026-10-27"

		return []*domain.ParsedMessage{
			validParsed(1, "ok"), notTransaction, badType, hugeAmount, future, validParsed(6, "db down"), yesterday,
		}, nil
	}}
	repo := &fakeTransactionRepo{failOn: "db down"}

	results, err := newTestRecordBatch(parser, repo).Execute(context.Background(), inputsFor(1, 1, 1, 1, 1, 1, 1))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tests := []struct {
		name    string
		wantErr error
	}{
		{name: "saves valid item"},
		{name: "rejects non transaction", wantErr: constant.ErrInvalidRequest},
		{name: "rejects unknown type", wantErr: constant.ErrInvalidTransactionType},
		{name: "rejects implausible amount", wantErr: constant.ErrInvalidTransactionAmount},
		{name: "rejects future date", wantErr: constant.ErrFutureTransactionDate},
		{name: "reports save failure", wantErr: constant.ErrInternal},
		{name: "saves backdated item"},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := results[i]
			if tt.wantErr == nil {
				if got.Err != nil || got.Transaction == nil {
					t.Fatalf("got %+v, want saved transaction", got)
				}
				return
			}
			if !errors.Is(got.Err, tt.wantErr) || got.Transaction != nil {
				t.Fatalf("got err %v, want %v", got.Err, tt.wantErr)
			}
		})
	}
}

func TestRecordBatchWholeBatchError(t *testing.T) {
	parser := &fakeBatchParser{respond: func(items []domain.ParseBatchItem) ([]*domain.ParsedMessage, error) {
		return nil, util.WrapStd(constant.ErrTooManyRequests, "quota", errors.New("429"))
	}}
	repo := &fakeTransactionRepo{}

	results, err := newTestRecordBatch(parser, repo).Execute(context.Background(), inputsFor(1, 2))
	if !errors.Is(err, constant.ErrTooManyRequests) || results != nil {
		t.Fatalf("got results %v err %v, want ErrTooManyRequests and no results", results, err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("created %d transactions, want 0", len(repo.created))
	}
}

func TestRecordBatchQuipSelection(t *testing.T) {
	parser := &fakeBatchParser{respond: echoAll}
	repo := &fakeTransactionRepo{}
	uc := NewRecordBatch(repo, parser) // real random picker

	userIDs := []int64{1, 2, 1, 3, 1, 2}
	for run := 0; run < 50; run++ {
		results, err := uc.Execute(context.Background(), inputsFor(userIDs...))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		askedPerUser := map[int64]int{}
		quipsPerUser := map[int64]int{}
		for i, item := range parser.received {
			if item.NeedQuip {
				askedPerUser[userIDs[i]]++
			}
			// the fake parser writes a quip for every item; only the asked one may survive
			if results[i].Quip != "" {
				quipsPerUser[userIDs[i]]++
				if !item.NeedQuip {
					t.Fatalf("item %d kept a quip it did not ask for", i)
				}
			}
		}

		for _, userID := range []int64{1, 2, 3} {
			if askedPerUser[userID] != 1 || quipsPerUser[userID] != 1 {
				t.Fatalf("user %d asked %d quips, got %d; want exactly 1", userID, askedPerUser[userID], quipsPerUser[userID])
			}
		}
	}
}
