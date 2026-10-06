package whatsapp

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestChatQueueKeepsOrderPerChat(t *testing.T) {
	q := newChatQueue(time.Minute)
	var mu sync.Mutex
	var got []int
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		i := i
		wg.Add(1)
		q.Enqueue("chat-a", func() {
			defer wg.Done()
			if i%7 == 0 {
				time.Sleep(time.Millisecond) // uneven job lengths
			}
			mu.Lock()
			got = append(got, i)
			mu.Unlock()
		})
	}
	wg.Wait()
	for i, v := range got {
		if v != i {
			t.Fatalf("jobs ran out of order: %v", got)
		}
	}
}

func TestChatQueueRunsChatsInParallel(t *testing.T) {
	q := newChatQueue(time.Minute)
	release := make(chan struct{})
	slowStarted := make(chan struct{})
	fastDone := make(chan struct{})

	q.Enqueue("slow-chat", func() {
		close(slowStarted)
		<-release // e.g. a typing pause or a slow Gemini call
	})
	<-slowStarted
	q.Enqueue("fast-chat", func() { close(fastDone) })

	select {
	case <-fastDone:
	case <-time.After(time.Second):
		t.Fatal("a slow chat blocked another chat")
	}
	close(release)
}

func TestChatQueueWorkerExitsWhenIdle(t *testing.T) {
	q := newChatQueue(20 * time.Millisecond)
	done := make(chan struct{})
	q.Enqueue("chat", func() { close(done) })
	<-done
	deadline := time.Now().Add(time.Second)
	for q.activeChats() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("idle worker never exited")
		}
		time.Sleep(5 * time.Millisecond)
	}

	// A new message after the worker exited starts a fresh one.
	again := make(chan struct{})
	q.Enqueue("chat", func() { close(again) })
	select {
	case <-again:
	case <-time.After(time.Second):
		t.Fatal("job after idle exit never ran")
	}
}

func TestChatQueueSurvivesPanic(t *testing.T) {
	q := newChatQueue(time.Minute)
	q.Enqueue("chat", func() { panic("boom") })
	done := make(chan struct{})
	q.Enqueue("chat", func() { close(done) })
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("queue stopped after a panicking job")
	}
}

func TestTypingDelay(t *testing.T) {
	short := typingDelay("Hi!")
	long := typingDelay(strings.Repeat("a", 500))
	if short < typingBase || short > time.Second {
		t.Errorf("short delay = %v", short)
	}
	if long != typingMax {
		t.Errorf("long delay = %v, want cap %v", long, typingMax)
	}
	if typingDelay("a somewhat longer quip about chicken") <= short {
		t.Error("longer text should take a bit longer")
	}
}
