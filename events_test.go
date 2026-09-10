package main

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestEventBrokerPublishesAndUnsubscribes(t *testing.T) {
	broker := newEventBroker()
	events, unsubscribe := broker.subscribe()
	broker.publish()
	select {
	case <-events:
	default:
		t.Fatal("subscriber did not receive event")
	}
	unsubscribe()
	broker.publish()
	if len(broker.subscribers) != 0 {
		t.Fatal("subscriber was not removed")
	}
}

func TestEventStreamHeadersAndInitialRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	response := httptest.NewRecorder()
	serveEvents(response, request, newEventBroker())

	if response.Header().Get("Content-Type") != "text/event-stream" || response.Header().Get("X-Accel-Buffering") != "no" {
		t.Fatalf("unexpected stream headers: %v", response.Header())
	}
	if !strings.Contains(response.Body.String(), "retry: 3000") {
		t.Fatalf("unexpected stream body: %q", response.Body.String())
	}
}
