package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
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
		_, _ = w.Write([]byte(`{"messageId":"webhook-message-1"}`))
	}))
	defer server.Close()

	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	provider := NewWebhookProvider(server.Client())
	result, err := provider.Deliver(t.Context(), data.NotificationDelivery{
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
	if result.ProviderMessageID != "webhook-message-1" {
		t.Fatalf("provider message id = %q", result.ProviderMessageID)
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

func TestWebhookProviderBoundsMessagePayload(t *testing.T) {
	var payload webhookPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	_, err := NewWebhookProvider(server.Client()).Deliver(t.Context(), data.NotificationDelivery{
		Target: server.URL,
		Title:  "  " + strings.Repeat("t", maxNotificationTitleLength+10) + "  ",
		Body:   "  " + strings.Repeat("b", maxNotificationBodyLength+10) + "  ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(payload.Title)) != maxNotificationTitleLength ||
		len([]rune(payload.Body)) != maxNotificationBodyLength {
		t.Fatalf("payload title/body lengths = %d/%d", len([]rune(payload.Title)), len([]rune(payload.Body)))
	}
	if strings.Contains(payload.Title, " ") || strings.Contains(payload.Body, " ") {
		t.Fatalf("payload text was not trimmed: %#v", payload)
	}
}

func TestNotificationTextBoundsOutboundText(t *testing.T) {
	title := "  " + strings.Repeat("t", maxNotificationTitleLength+10) + "  "
	body := "  " + strings.Repeat("b", maxNotificationBodyLength+10) + "  "

	if got := notificationTitle(title); len([]rune(got)) != maxNotificationTitleLength || strings.Contains(got, " ") {
		t.Fatalf("notificationTitle length/text = %d/%q", len([]rune(got)), got)
	}
	if got := notificationBody(body); len([]rune(got)) != maxNotificationBodyLength || strings.Contains(got, " ") {
		t.Fatalf("notificationBody length/text = %d/%q", len([]rune(got)), got)
	}
	text := notificationText(title, body)
	if len([]rune(text)) != maxNotificationTextLength {
		t.Fatalf("notificationText length = %d, want %d", len([]rune(text)), maxNotificationTextLength)
	}
	if !strings.HasPrefix(text, strings.Repeat("t", maxNotificationTitleLength)+"\n\n") {
		t.Fatalf("notificationText prefix = %q", text[:maxNotificationTitleLength+2])
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
	_, err := provider.Deliver(t.Context(), data.NotificationDelivery{
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
	_, err := provider.Deliver(t.Context(), data.NotificationDelivery{
		Target: "demo://ops",
	})
	if err == nil {
		t.Fatal("expected invalid webhook target error")
	}
}

func TestValidateProviderTarget(t *testing.T) {
	t.Setenv("TELEGRAM_TEST_TOKEN", "123456:abcdef")
	t.Setenv("FEISHU_TEST_WEBHOOK", "https://example.test/feishu")
	t.Setenv("SMTP_TEST_USER", "bot@example.com")
	t.Setenv("SMTP_TEST_PASSWORD", "smtp-secret")

	tests := []struct {
		name     string
		provider string
		target   string
	}{
		{name: "local", provider: "local", target: "ops"},
		{name: "webhook", provider: "webhook", target: "https://example.test/hook"},
		{name: "telegram", provider: "telegram", target: "telegram://send?chat_id=1&token_env=TELEGRAM_TEST_TOKEN"},
		{name: "feishu", provider: "feishu", target: "feishu://webhook?url_env=FEISHU_TEST_WEBHOOK"},
		{name: "email", provider: "email", target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&username_env=SMTP_TEST_USER&password_env=SMTP_TEST_PASSWORD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProviderTarget(tt.provider, tt.target); err != nil {
				t.Fatalf("ValidateProviderTarget error = %v", err)
			}
		})
	}
}

func TestValidateProviderTargetRejectsInvalidTarget(t *testing.T) {
	err := ValidateProviderTarget("telegram", "telegram://send?chat_id=1&token_env=MISSING_TOKEN")
	if err == nil {
		t.Fatal("expected invalid target error")
	}
}

func TestValidateProviderTargetSyntaxAllowsUnsetEnvReferences(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		target   string
	}{
		{name: "local failure simulation", provider: "local", target: "fail-delivery"},
		{name: "telegram", provider: "telegram", target: "telegram://send?chat_id=1&token_env=UNSET_TELEGRAM_TOKEN"},
		{name: "feishu", provider: "feishu", target: "feishu://webhook?url_env=UNSET_FEISHU_URL"},
		{name: "email", provider: "email", target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&username_env=UNSET_SMTP_USER&password_env=UNSET_SMTP_PASSWORD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProviderTargetSyntax(tt.provider, tt.target); err != nil {
				t.Fatalf("ValidateProviderTargetSyntax error = %v", err)
			}
		})
	}
}

func TestValidateProviderTargetSyntaxRejectsInvalidTarget(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		target   string
	}{
		{name: "webhook scheme", provider: "webhook", target: "ftp://example.test/hook"},
		{name: "telegram missing token env", provider: "telegram", target: "telegram://send?chat_id=1"},
		{name: "telegram invalid token env", provider: "telegram", target: "telegram://send?chat_id=1&token_env=1BAD"},
		{name: "feishu invalid url env", provider: "feishu", target: "feishu://webhook?url_env=BAD-NAME"},
		{name: "email password without username", provider: "email", target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&password_env=SMTP_PASSWORD"},
		{name: "email starttls", provider: "email", target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&starttls=maybe"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateProviderTargetSyntax(tt.provider, tt.target); err == nil {
				t.Fatal("expected invalid target error")
			}
		})
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
		_, err := provider.Deliver(ctx, data.NotificationDelivery{Target: server.URL})
		errCh <- err
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
