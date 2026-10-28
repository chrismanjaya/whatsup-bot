package service

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"
	"sync/atomic"
	"time"
	"whatsupbot/internal/constant"
	"whatsupbot/internal/usecase/render"
	transactionuc "whatsupbot/internal/usecase/transaction"
	"whatsupbot/internal/util"
)

const (
	// ackSkipWindow: when the message's result is due within this long, the canned ack is skipped since the confirmation follows right away.
	ackSkipWindow = 5 * time.Second
)

type (
	RecordBatcherConfig struct {
		FlushInterval time.Duration
		MaxBatchSize  int
	}

	// RecordDelivery is everything the delivery adapter needs to answer one finished record message.
	RecordDelivery struct {
		Message RecordMessage
		Replies []render.Reply
	}

	recordBatchExecutor interface {
		Execute(ctx context.Context, inputs []transactionuc.RecordBatchInput) ([]transactionuc.RecordBatchResult, error)
	}

	errorMessagePicker interface {
		Pick(jid string, errCode *constant.ErrCode) string
	}

	// RecordBatcher queues record messages from every user and parses them together, one parser call per tick, to stay within the parser's rate limit.
	RecordBatcher struct {
		queue       *recordQueue
		recordBatch recordBatchExecutor
		picker      errorMessagePicker
		config      RecordBatcherConfig

		now         func() time.Time
		nextFlushAt atomic.Int64 // unix nanos of the next tick; 0 until Run starts
	}
)

func NewRecordBatcher(
	recordBatch recordBatchExecutor,
	picker errorMessagePicker,
	config RecordBatcherConfig,
) *RecordBatcher {
	return &RecordBatcher{
		queue:       newRecordQueue(),
		recordBatch: recordBatch,
		picker:      picker,
		config:      config,
		now:         time.Now,
	}
}

// Enqueue adds message to the shared queue. It reports whether anything was queued and whether the user should get a canned ack: at most once per batch window, and not when the confirmation is due within ackSkipWindow anyway.
func (b *RecordBatcher) Enqueue(message RecordMessage) (bool, bool) {
	pushed := b.queue.push(message)
	if !pushed.isQueued {
		return false, false
	}

	return true, pushed.isFirstAck && !b.isResultImminent(pushed.queueLen)
}

// isResultImminent reports whether the next tick is within ackSkipWindow and will include every item up to queueLen.
func (b *RecordBatcher) isResultImminent(queueLen int) bool {
	nextFlushAt := b.nextFlushAt.Load()
	if nextFlushAt == 0 || queueLen > b.config.MaxBatchSize {
		return false
	}

	return time.Unix(0, nextFlushAt).Sub(b.now()) < ackSkipWindow
}

func (b *RecordBatcher) scheduleNextFlush() {
	b.nextFlushAt.Store(b.now().Add(b.config.FlushInterval).UnixNano())
}

// Run flushes the queue every FlushInterval until ctx is done, handing each finished message to deliver.
func (b *RecordBatcher) Run(ctx context.Context, deliver func(context.Context, RecordDelivery)) {
	ticker := time.NewTicker(b.config.FlushInterval)
	defer ticker.Stop()
	b.scheduleNextFlush()

	for {
		select {
		case <-ctx.Done():
			if pending := b.queue.len(); pending > 0 {
				slog.Warn("record batcher stopped with pending items", "pending", pending)
			}
			return
		case <-ticker.C:
			b.scheduleNextFlush()
			b.safeFlush(ctx, deliver)
		}
	}
}

// safeFlush logs a panic instead of killing the ticker loop.
func (b *RecordBatcher) safeFlush(ctx context.Context, deliver func(context.Context, RecordDelivery)) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic", "where", "record batcher flush", "panic", r, "stack", string(debug.Stack()))
		}
	}()
	b.flush(ctx, deliver)
}

func (b *RecordBatcher) flush(ctx context.Context, deliver func(context.Context, RecordDelivery)) {
	// take the oldest items
	batch := b.queue.take(b.config.MaxBatchSize)
	if len(batch) == 0 {
		return
	}

	// build inputs; position i maps back to batch[i] (user, message, line)
	inputs := make([]transactionuc.RecordBatchInput, 0, len(batch))
	for _, item := range batch {
		inputs = append(inputs, transactionuc.RecordBatchInput{
			UserID:     item.entry.message.UserID,
			GroupID:    item.entry.message.GroupID,
			Text:       item.entry.lines[item.line],
			ReceivedAt: item.entry.message.ReceivedAt,
		})
	}

	// parse and save batch
	results, err := b.recordBatch.Execute(ctx, inputs)
	if err == nil && len(results) != len(batch) {
		err = util.Wrap(constant.ErrInternal, "record batch result count mismatch")
	}

	if err != nil {
		// rate limited or unavailable: keep every item for the next tick
		if errors.Is(err, constant.ErrTooManyRequests) || errors.Is(err, constant.ErrServiceUnavailable) {
			b.queue.requeue(batch)
			util.LogWarn("record batch deferred to next tick", err, "items", len(batch))
			return
		}

		util.LogError("record batch failed", err, "items", len(batch))
		results = make([]transactionuc.RecordBatchResult, len(batch))
		for i := range results {
			results[i] = transactionuc.RecordBatchResult{Err: err}
		}
	}

	// assign results and deliver finished messages
	for i, item := range batch {
		item.entry.results[item.line] = b.toLineResult(item, results[i])
		item.entry.remaining--
		if item.entry.remaining == 0 {
			deliver(ctx, RecordDelivery{
				Message: item.entry.message,
				Replies: render.NewRecordResponse(ctx, item.entry.results).Render(),
			})
		}
	}
}

func (b *RecordBatcher) toLineResult(item *recordItem, result transactionuc.RecordBatchResult) render.RecordLineResult {
	line := render.RecordLineResult{
		Text:        item.entry.lines[item.line],
		Transaction: result.Transaction,
		Quip:        result.Quip,
	}

	if result.Err != nil || result.Transaction == nil {
		util.LogError("record item failed", result.Err, "user_id", item.entry.message.UserID, "message_id", item.entry.message.MessageID)
		errCode := mappingResponseCode(result.Err)
		if errCode == nil {
			errCode = constant.ErrInternal
		}
		line.Transaction = nil
		line.ErrorMessage = b.picker.Pick(item.entry.message.SenderJID, errCode)
	}

	return line
}
