package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestVisionClientLimitsConcurrentAnalysis(t *testing.T) {
	started := make(chan struct{}, 3)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		started <- struct{}{}
		<-release
		writeJSON(w, http.StatusOK, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"valid":true,"score":1,"reason":""}`}}}})
	}))
	defer server.Close()
	client := newVisionClient(serviceConfig{URL: server.URL, Model: "test", APIKey: "key"})
	results := make(chan error, 3)
	for range 3 {
		go func() {
			_, err := client.analyze(context.Background(), gameConfig{Name: "Test"}, "image/png", testPNG)
			results <- err
		}()
	}
	<-started
	<-started
	select {
	case <-started:
		t.Fatal("more than two analyses ran concurrently")
	case <-time.After(50 * time.Millisecond):
	}
	close(release)
	for range 3 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
}
