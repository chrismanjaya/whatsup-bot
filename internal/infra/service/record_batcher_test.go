package service

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/domain"
	"whatsupbot/internal/usecase/render"
	transactionuc "whatsupbot/internal/usecase/transaction"
	"whatsupbot/internal/util"
)

func TestMain(m *testing.M) {
	util.InitTemplate()
	os.Exit(m.Run())
}

// fakeRecordBatch saves every input whose text does not start with "bad", unless err is set.
type fakeRecordBatch struct {
	calls [][]transactionuc.RecordBatchInput
	err   error
}

func (f *fakeRecordBatch) Execute(ctx context.Context, inputs []transactionuc.RecordBatchInput) ([]transactionuc.RecordBatchResult, error) {
	f.calls = append(f.calls, inputs)
	if f.err != nil {
		return nil, f.err
	}

	results := make([]transactionuc.RecordBatchResult, len(inputs))
	for i, input := range inputs {
		if strings.HasPrefix(input.Text, "bad") {
			results[i] = transactionuc.RecordBatchResult{Err: util.Wrap(constant.ErrInvalidRequest, "not a transaction")}
			continue
		}
		results[i] = transactionuc.RecordBatchResult{Transaction: &domain.Transaction{
			ID:              int64(i + 1),
			UserID:          input.UserID,
			Description:     input.Text,
			Type:            domain.Expense,
			Category:        domain.CategoryFood,
			TransactionDate: input.ReceivedAt,
		}}
	}
	return results, nil
}

// fakePicker mimics persona.Picker: an emoji per error code, never the code itself.
type fakePicker struct{}

func (fakePicker) Pick(jid string, errCode *constant.ErrCode) string {
	return map[int]string{400: "🧪 not money", 500: "🥱 lab broke"}[errCode.Code]
}

type deliveries []RecordDelivery

func (d *deliveries) deliver(ctx context.Context, delivery RecordDelivery) {
	*d = append(*d, delivery)
}

func newTestBatcher(executor *fakeRecordBatch, maxBatchSize int) *RecordBatcher {
	return NewRecordBatcher(executor, fakePicker{}, RecordBatcherConfig{
		FlushInterval: time.Hour,
		MaxBatchSize:  maxBatchSize,
	})
}

func recordMessage(userID int64, messageID, text string) RecordMessage {
	return RecordMessage{
		UserID:     userID,
		MessageID:  messageID,
		ChatJID:    "chat-" + messageID,
		SenderJID:  "sender-" + messageID,
		Text:       text,
		ReceivedAt: time.Date(2026, 10, 28, 9, 0, 0, 0, util.Jakarta),
	}
}

func inputTexts(inputs []transactionuc.RecordBatchInput) []string {
	texts := make([]string, 0, len(inputs))
	for _, input := range inputs {
		texts = append(texts, input.Text)
	}
	return texts
}

func TestRecordBatcherFlushMapsResultsToMessages(t *testing.T) {
	executor := &fakeRecordBatch{}
	batcher := newTestBatcher(executor, 10)
	var got deliveries

	batcher.Enqueue(recordMessage(1, "m1", "kopi 25k"))
	batcher.Enqueue(recordMessage(2, "m2", "bensin 50k"))
	batcher.flush(context.Background(), got.deliver)

	if len(executor.calls) != 1 || len(executor.calls[0]) != 2 {
		t.Fatalf("got %d parser calls, want one call with both users", len(executor.calls))
	}
	if len(got) != 2 {
		t.Fatalf("got %d deliveries, want 2", len(got))
	}

	for i, want := range []struct {
		userID    int64
		messageID string
		text      string
	}{{1, "m1", "kopi 25k"}, {2, "m2", "bensin 50k"}} {
		delivery := got[i]
		if delivery.Message.UserID != want.userID || delivery.Message.MessageID != want.messageID {
			t.Fatalf("delivery %d went to user %d message %s, want user %d message %s",
				i, delivery.Message.UserID, delivery.Message.MessageID, want.userID, want.messageID)
		}

		replies := delivery.Replies
		if len(replies) != 2 || replies[0].Reaction != render.ReactionSuccess || !replies[1].IsQuoted {
			t.Fatalf("delivery %d replies = %+v, want ✅ then quoted confirmation", i, replies)
		}
		transaction := replies[1].Metadata.Transaction
		if transaction == nil || transaction.UserID != want.userID || transaction.Description != want.text {
			t.Fatalf("delivery %d transaction = %+v, want user %d %q", i, transaction, want.userID, want.text)
		}
	}

	if batcher.queue.len() != 0 {
		t.Fatalf("queue has %d items after flush, want 0", batcher.queue.len())
	}
}

func TestRecordBatcherRequeuesOnRateLimit(t *testing.T) {
	for _, tt := range []struct {
		name string
		err  *constant.ErrCode
	}{
		{name: "too many requests", err: constant.ErrTooManyRequests},
		{name: "service unavailable after retries", err: constant.ErrServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			executor := &fakeRecordBatch{err: util.WrapStd(tt.err, "gemini failed", errors.New("boom"))}
			batcher := newTestBatcher(executor, 10)
			var got deliveries

			batcher.Enqueue(recordMessage(1, "m1", "kopi 25k"))
			batcher.flush(context.Background(), got.deliver)

			// arrives while the failed batch is in flight; must stay behind it
			batcher.Enqueue(recordMessage(2, "m2", "bensin 50k"))

			if len(got) != 0 {
				t.Fatalf("got %d deliveries, want none while rate limited", len(got))
			}

			executor.err = nil
			batcher.flush(context.Background(), got.deliver)

			if want := []string{"kopi 25k", "bensin 50k"}; !reflect.DeepEqual(inputTexts(executor.calls[1]), want) {
				t.Fatalf("retry sent %v, want %v", inputTexts(executor.calls[1]), want)
			}
			if len(got) != 2 || got[0].Message.MessageID != "m1" {
				t.Fatalf("got %d deliveries, want m1 then m2", len(got))
			}
		})
	}
}

func TestRecordBatcherFailsItemsOnOtherErrors(t *testing.T) {
	executor := &fakeRecordBatch{err: util.WrapStd(constant.ErrInternal, "bad json", errors.New("boom"))}
	batcher := newTestBatcher(executor, 10)
	var got deliveries

	batcher.Enqueue(recordMessage(1, "m1", "kopi 25k"))
	batcher.flush(context.Background(), got.deliver)

	if len(got) != 1 || got[0].Replies[0].Reaction != render.ReactionFailed {
		t.Fatalf("got %+v, want one ⚠️ delivery", got)
	}
	if text := got[0].Replies[1].Text; text != "🥱 lab broke" || !got[0].Replies[1].IsQuoted {
		t.Fatalf("error reply %q should be the quoted persona message", text)
	}
	if batcher.queue.len() != 0 {
		t.Fatalf("queue has %d items, want failed items dropped", batcher.queue.len())
	}
}

func TestRecordBatcherMaxBatchSizeTakesOldestFirst(t *testing.T) {
	executor := &fakeRecordBatch{}
	batcher := newTestBatcher(executor, 2)
	var got deliveries

	batcher.Enqueue(recordMessage(1, "m1", "a"))
	batcher.Enqueue(recordMessage(2, "m2", "b"))
	batcher.Enqueue(recordMessage(3, "m3", "c"))

	batcher.flush(context.Background(), got.deliver)
	if want := []string{"a", "b"}; !reflect.DeepEqual(inputTexts(executor.calls[0]), want) {
		t.Fatalf("first batch = %v, want %v", inputTexts(executor.calls[0]), want)
	}
	if batcher.queue.len() != 1 {
		t.Fatalf("queue has %d items, want 1 left for next tick", batcher.queue.len())
	}

	batcher.flush(context.Background(), got.deliver)
	if want := []string{"c"}; !reflect.DeepEqual(inputTexts(executor.calls[1]), want) {
		t.Fatalf("second batch = %v, want %v", inputTexts(executor.calls[1]), want)
	}
	if len(got) != 3 {
		t.Fatalf("got %d deliveries, want 3", len(got))
	}
}

func TestRecordBatcherMultiLineMessage(t *testing.T) {
	executor := &fakeRecordBatch{}
	batcher := newTestBatcher(executor, 2) // forces the 3 lines across two batches
	var got deliveries

	batcher.Enqueue(recordMessage(1, "m1", "kopi 25k\n\n  bad line  \nbensin 50k"))

	batcher.flush(context.Background(), got.deliver)
	if len(got) != 0 {
		t.Fatalf("delivered before every line was parsed")
	}

	batcher.flush(context.Background(), got.deliver)
	if len(got) != 1 {
		t.Fatalf("got %d deliveries, want one summary for the message", len(got))
	}

	// one ⚠️ reaction, then one quoted reply per line, in line order
	replies := got[0].Replies
	if len(replies) != 4 || replies[0].Reaction != render.ReactionFailed {
		t.Fatalf("replies = %+v, want ⚠️ then 3 quoted replies", replies)
	}
	for i, reply := range replies[1:] {
		if !reply.IsQuoted {
			t.Fatalf("reply %d is not quoted to the original message", i+1)
		}
	}

	for i, want := range []string{"kopi 25k", "bensin 50k"} {
		reply := replies[[]int{1, 3}[i]]
		transaction := reply.Metadata.Transaction
		if transaction == nil || transaction.Description != want || !strings.Contains(reply.Text, "*EXPENSE*") {
			t.Fatalf("reply %+v should be the editable confirmation for %q", reply, want)
		}
	}

	failed := replies[2]
	if failed.Metadata.Transaction != nil || !strings.Contains(failed.Text, `"bad line"`) || !strings.Contains(failed.Text, "🧪 not money") {
		t.Fatalf("failed reply %q should name the line and use the persona message", failed.Text)
	}
}

func TestRecordBatcherAckOncePerWindow(t *testing.T) {
	batcher := newTestBatcher(&fakeRecordBatch{}, 10)
	var got deliveries

	steps := []struct {
		name    string
		userID  int64
		flush   bool
		wantAck bool
	}{
		{name: "first message acks", userID: 1, wantAck: true},
		{name: "second message in window is silent", userID: 1, wantAck: false},
		{name: "other user acks", userID: 2, wantAck: true},
		{name: "new window acks again", userID: 1, flush: true, wantAck: true},
	}

	for _, step := range steps {
		if step.flush {
			batcher.flush(context.Background(), got.deliver)
		}
		isQueued, needAck := batcher.Enqueue(recordMessage(step.userID, "m", "kopi"))
		if !isQueued || needAck != step.wantAck {
			t.Fatalf("%s: queued=%v ack=%v, want ack=%v", step.name, isQueued, needAck, step.wantAck)
		}
	}
}

func TestRecordBatcherSkipsAckWhenResultIsImminent(t *testing.T) {
	now := time.Date(2026, 10, 28, 9, 0, 0, 0, util.Jakarta)

	tests := []struct {
		name       string
		untilFlush time.Duration // 0 means Run has not started
		queued     int           // lines already waiting
		text       string
		wantAck    bool
	}{
		{name: "acks when the flush is far", untilFlush: 10 * time.Second, text: "kopi", wantAck: true},
		{name: "skips ack when the flush is near", untilFlush: 3 * time.Second, text: "kopi", wantAck: false},
		{name: "acks when the queue overflows the next batch", untilFlush: 3 * time.Second, queued: 3, text: "kopi", wantAck: true},
		{name: "acks when a multi-line message overflows the next batch", untilFlush: 3 * time.Second, text: "a\nb\nc\nd", wantAck: true},
		{name: "acks before the ticker starts", text: "kopi", wantAck: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batcher := newTestBatcher(&fakeRecordBatch{}, 3)
			batcher.now = func() time.Time { return now }
			if tt.untilFlush > 0 {
				batcher.nextFlushAt.Store(now.Add(tt.untilFlush).UnixNano())
			}
			for i := 0; i < tt.queued; i++ {
				batcher.Enqueue(recordMessage(99, "other", "teh"))
			}

			if _, needAck := batcher.Enqueue(recordMessage(1, "m1", tt.text)); needAck != tt.wantAck {
				t.Fatalf("ack = %v, want %v", needAck, tt.wantAck)
			}
		})
	}
}

func TestSplitRecordLines(t *testing.T) {
	tests := []struct {
		name string
		text string
		want []string
	}{
		{name: "single line", text: "kopi 25k", want: []string{"kopi 25k"}},
		{name: "splits and trims lines", text: " kopi 25k \nbensin 50k\r\n parkir 5k", want: []string{"kopi 25k", "bensin 50k", "parkir 5k"}},
		{name: "drops blank lines", text: "kopi\n\n   \nteh", want: []string{"kopi", "teh"}},
		{name: "blank text has no lines", text: " \n ", want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitRecordLines(tt.text); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitRecordLines(%q) = %q, want %q", tt.text, got, tt.want)
			}
		})
	}
}
