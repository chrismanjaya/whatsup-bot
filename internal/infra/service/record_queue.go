package service

import (
	"strings"
	"sync"
	"time"
	"whatsupbot/internal/usecase/render"
)

type (
	// RecordMessage is one incoming record message from an already validated user, waiting for the batch parser.
	RecordMessage struct {
		UserID     int64
		GroupID    *int64
		MessageID  string
		ChatJID    string
		SenderJID  string
		Text       string
		ReceivedAt time.Time
	}

	// recordEntry tracks a message whose lines may be parsed across several batches; it is delivered once remaining reaches 0.
	recordEntry struct {
		message   RecordMessage
		lines     []string
		results   []render.RecordLineResult
		remaining int
	}

	// recordItem is one line of an entry: the unit sent to the parser.
	recordItem struct {
		entry *recordEntry
		line  int
	}

	// recordQueue is the single, global FIFO of record lines shared by every user and chat.
	recordQueue struct {
		mu         sync.Mutex
		items      []*recordItem
		ackedUsers map[int64]struct{} // users sent a canned ack since the last flush
	}
)

func newRecordQueue() *recordQueue {
	return &recordQueue{
		ackedUsers: map[int64]struct{}{},
	}
}

// push splits message into lines and appends them. It reports whether the user should get a canned ack: at most once per batch window. A message with no non-blank line is ignored.
func (q *recordQueue) push(message RecordMessage) (bool, bool) {
	lines := splitRecordLines(message.Text)
	if len(lines) == 0 {
		return false, false
	}

	entry := &recordEntry{
		message:   message,
		lines:     lines,
		results:   make([]render.RecordLineResult, len(lines)),
		remaining: len(lines),
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	for i := range lines {
		q.items = append(q.items, &recordItem{entry: entry, line: i})
	}

	_, acked := q.ackedUsers[message.UserID]
	q.ackedUsers[message.UserID] = struct{}{}

	return true, !acked
}

// take removes and returns up to max of the oldest items, and opens a new ack window.
func (q *recordQueue) take(max int) []*recordItem {
	q.mu.Lock()
	defer q.mu.Unlock()

	clear(q.ackedUsers)

	n := min(max, len(q.items))
	if n <= 0 {
		return nil
	}

	batch := make([]*recordItem, n)
	copy(batch, q.items[:n])
	q.items = q.items[n:]

	return batch
}

// requeue puts items back at the front, keeping them ahead of anything that arrived since take.
func (q *recordQueue) requeue(items []*recordItem) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.items = append(append(make([]*recordItem, 0, len(items)+len(q.items)), items...), q.items...)
}

func (q *recordQueue) len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.items)
}

// splitRecordLines treats every non-blank line as its own transaction.
func splitRecordLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
