package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestWebhookProviderPostsJSONPayload(t *testing.T) {
	var payload webhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	provider := NewWebhookProvider(server.Client())
	err := provider.Deliver(t.Context(), data.NotificationDelivery{
		ID:             "no_1",
		NotificationID: "nt_1",
		TaskID:         "tt_1",
		IntentID:       "si_1",
		Channel:        "ops",
		Target:         server.URL,
		Title:          "Strategy intent",
		Body:           "buy signal",
		AttemptCount:   2,
		MaxAttempts:    3,
		CreatedAt:      createdAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload.NotificationID != "nt_1" || payload.DeliveryID != "no_1" {
		t.Fatalf("unexpected ids: %#v", payload)
	}
	if payload.Title != "Strategy intent" || payload.Body != "buy signal" {
		t.Fatalf("unexpected message payload: %#v", payload)
	}
	if payload.AttemptCount != 2 || payload.MaxAttempts != 3 {
		t.Fatalf("unexpected attempts: %#v", payload)
	}
	if !payload.CreatedAt.Equal(createdAt) {
		t.Fatalf("createdAt = %v, want %v", payload.CreatedAt, createdAt)
	}
}

func TestWebhookProviderRejectsNonHTTPURL(t *testing.T) {
	provider := NewWebhookProvider(nil)
	err := provider.Deliver(t.Context(), data.NotificationDelivery{
		Target: "demo://ops",
	})
	if err == nil {
		t.Fatal("expected invalid webhook target error")
	}
}

func TestWebhookProviderUsesContextCancellation(t *testing.T) {
	started := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(500 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	errCh := make(chan error, 1)
	go func() {
		provider := NewWebhookProvider(server.Client())
		errCh <- provider.Deliver(ctx, data.NotificationDelivery{Target: server.URL})
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("webhook request did not reach the server")
	}

	select {
	case err := <-errCh:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("error = %v, want context deadline exceeded", err)
		}
	case <-time.After(time.Second):
		t.Fatal("webhook delivery did not return after context cancellation")
	}
}
