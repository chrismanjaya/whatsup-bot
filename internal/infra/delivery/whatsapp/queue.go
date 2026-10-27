package whatsapp

import (
	"log/slog"
	"sync"
	"time"
)

const (
	// How many messages one chat can have waiting before Enqueue blocks.
	chatQueueBuffer = 64
)

type chatQueue struct {
	mu      sync.Mutex
	workers map[string]chan func()
	idle    time.Duration
}

func newChatQueue(idle time.Duration) *chatQueue {
	return &chatQueue{
		workers: map[string]chan func(){},
		idle:    idle,
	}
}

func (q *chatQueue) enqueue(chat string, job func()) {
	q.mu.Lock()
	defer q.mu.Unlock()

	jobs, exists := q.workers[chat]
	if !exists {
		jobs = make(chan func(), chatQueueBuffer)
		q.workers[chat] = jobs
		go q.run(chat, jobs)
	}
	// Sending while holding mu means the worker can't decide to exit between us finding it and handing it the job.
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

// Runs job, logging a panic instead of crashing the whole bot.
func runJob(chat string, job func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("chat job panicked", "chat", chat, "panic", r)
		}
	}()
	job()
}
