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
	var requestID string
	var traceparent string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("content-type = %q", r.Header.Get("Content-Type"))
		}
		requestID = r.Header.Get(outboundRequestIDHeader)
		traceparent = r.Header.Get(outboundTraceParentHeader)
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
		RequestID:      "request-id-webhook",
		TraceParent:    "00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
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
	if requestID != "request-id-webhook" || payload.RequestID != "request-id-webhook" {
		t.Fatalf("request id header = %q payload = %q", requestID, payload.RequestID)
	}
	if traceparent != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" ||
		payload.TraceParent != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("traceparent header = %q payload = %q", traceparent, payload.TraceParent)
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

func TestWebhookProviderSkipsUnsafeRequestIDHeader(t *testing.T) {
	var requestID string
	var traceparent string
	var payload webhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID = r.Header.Get(outboundRequestIDHeader)
		traceparent = r.Header.Get(outboundTraceParentHeader)
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	provider := NewWebhookProvider(server.Client())
	err := provider.Deliver(t.Context(), data.NotificationDelivery{
		RequestID:   "bad\nrequest",
		TraceParent: "00-00000000000000000000000000000000-00f067aa0ba902b7-01",
		Target:      server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if requestID != "" || payload.RequestID != "" {
		t.Fatalf("unsafe request id header = %q payload = %q", requestID, payload.RequestID)
	}
	if traceparent != "" || payload.TraceParent != "" {
		t.Fatalf("unsafe traceparent header = %q payload = %q", traceparent, payload.TraceParent)
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
