package main

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

type eventBroker struct {
	mu          sync.Mutex
	subscribers map[chan struct{}]struct{}
	done        chan struct{}
	stop        sync.Once
}

func newEventBroker() *eventBroker {
	return &eventBroker{subscribers: map[chan struct{}]struct{}{}, done: make(chan struct{})}
}

func (broker *eventBroker) close() {
	broker.stop.Do(func() { close(broker.done) })
}

func (broker *eventBroker) subscribe() (<-chan struct{}, func()) {
	channel := make(chan struct{}, 1)
	broker.mu.Lock()
	broker.subscribers[channel] = struct{}{}
	broker.mu.Unlock()
	return channel, func() {
		broker.mu.Lock()
		delete(broker.subscribers, channel)
		broker.mu.Unlock()
	}
}

func (broker *eventBroker) publish() {
	broker.mu.Lock()
	defer broker.mu.Unlock()
	for subscriber := range broker.subscribers {
		select {
		case subscriber <- struct{}{}:
		default:
		}
	}
}

func serveEvents(w http.ResponseWriter, r *http.Request, broker *eventBroker) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	events, unsubscribe := broker.subscribe()
	defer unsubscribe()
	fmt.Fprint(w, "retry: 3000\n\n")
	flusher.Flush()
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-broker.done:
			return
		case <-events:
			fmt.Fprint(w, "event: leaderboard\ndata: changed\n\n")
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": keep-alive\n\n")
			flusher.Flush()
		}
	}
}
