package whatsapp

import (
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"
)

// chatQueue runs jobs one at a time per chat, in arrival order, while
// different chats run in parallel. whatsmeow calls event handlers in one
// serial loop, so handling a message inline (Gemini call, typing pause)
// would make every other chat wait; queueing per chat keeps a slow reply in
// one chat from delaying the others, and keeps replies within a chat in
// order.
//
// Each chat gets a worker goroutine on its first message, which exits after
// idle with no new jobs, so quiet chats don't hold a goroutine.
type chatQueue struct {
	mu      sync.Mutex
	workers map[string]chan func()
	idle    time.Duration
}

// chatQueueBuffer is how many messages one chat can have waiting before
// Enqueue blocks.
const chatQueueBuffer = 64

func newChatQueue(idle time.Duration) *chatQueue {
	return &chatQueue{workers: map[string]chan func(){}, idle: idle}
}

// Enqueue schedules job to run after the chat's earlier jobs. It returns
// right away unless the chat already has chatQueueBuffer jobs waiting.
func (q *chatQueue) Enqueue(chat string, job func()) {
	q.mu.Lock()
	defer q.mu.Unlock()
	jobs, ok := q.workers[chat]
	if !ok {
		jobs = make(chan func(), chatQueueBuffer)
		q.workers[chat] = jobs
		go q.run(chat, jobs)
	}
	// Sending while holding mu means the worker can't decide to exit
	// between us finding it and handing it the job.
	jobs <- job
}

func (q *chatQueue) run(chat string, jobs chan func()) {
	timer := time.NewTimer(q.idle)
	defer timer.Stop()
	for {
		select {
		case job := <-jobs:
			runJob(chat, job)
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(q.idle)
		case <-timer.C:
			q.mu.Lock()
			if len(jobs) == 0 {
				delete(q.workers, chat)
				q.mu.Unlock()
				return
			}
			q.mu.Unlock()
			timer.Reset(q.idle)
		}
	}
}

// runJob runs job, logging a panic instead of crashing the whole bot.
func runJob(chat string, job func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("chat job panicked", "chat", chat, "panic", r)
		}
	}()
	job()
}

// activeChats returns how many chats currently have a worker (for tests).
func (q *chatQueue) activeChats() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.workers)
}

// Typing pause before a chatty reply (Reply.Typing): long enough to see
// "typing...", scaled a little by length, capped so it never drags.
const (
	typingBase    = 700 * time.Millisecond
	typingPerRune = 20 * time.Millisecond
	typingMax     = 2 * time.Second
)

func typingDelay(text string) time.Duration {
	d := typingBase + time.Duration(utf8.RuneCountInString(text))*typingPerRune
	if d > typingMax {
		return typingMax
	}
	return d
}
