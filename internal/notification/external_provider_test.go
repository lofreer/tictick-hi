package notification

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestTelegramProviderPostsSendMessagePayload(t *testing.T) {
	t.Setenv("TELEGRAM_TEST_TOKEN", "123456:telegram-secret")

	var payload telegramPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/bot123456:telegram-secret/sendMessage" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewTelegramProvider(server.Client())
	err := provider.Deliver(t.Context(), data.NotificationDelivery{
		Target: "telegram://send?chat_id=ops-chat&token_env=TELEGRAM_TEST_TOKEN&api_base=" + server.URL,
		Title:  "Signal",
		Body:   "Buy BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload.ChatID != "ops-chat" || payload.Text != "Signal\n\nBuy BTCUSDT" {
		t.Fatalf("unexpected telegram payload: %#v", payload)
	}
}

func TestTelegramProviderRedactsTokenFromErrors(t *testing.T) {
	t.Setenv("TELEGRAM_TEST_TOKEN", "123456:telegram-secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "bad token 123456:telegram-secret", http.StatusUnauthorized)
	}))
	defer server.Close()

	err := NewTelegramProvider(server.Client()).Deliver(t.Context(), data.NotificationDelivery{
		Target: "telegram://send?chat_id=ops-chat&token_env=TELEGRAM_TEST_TOKEN&api_base=" + server.URL,
		Title:  "Signal",
	})
	if err == nil {
		t.Fatal("expected telegram error")
	}
	if strings.Contains(err.Error(), "telegram-secret") {
		t.Fatalf("telegram error leaked token: %v", err)
	}
}

func TestFeishuProviderPostsTextMessage(t *testing.T) {
	var payload feishuPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	t.Setenv("FEISHU_TEST_WEBHOOK", server.URL+"/open-apis/bot/v2/hook/secret")

	err := NewFeishuProvider(server.Client()).Deliver(t.Context(), data.NotificationDelivery{
		Target: "feishu://webhook?url_env=FEISHU_TEST_WEBHOOK",
		Title:  "Signal",
		Body:   "Buy BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if payload.MessageType != "text" || payload.Content.Text != "Signal\n\nBuy BTCUSDT" {
		t.Fatalf("unexpected feishu payload: %#v", payload)
	}
}

func TestEmailProviderBuildsSMTPMessageFromEnvironment(t *testing.T) {
	t.Setenv("SMTP_TEST_USER", "bot@example.com")
	t.Setenv("SMTP_TEST_PASSWORD", "smtp-secret")
	sender := &captureMailSender{}
	provider := NewEmailProvider(sender)

	err := provider.Deliver(t.Context(), data.NotificationDelivery{
		Target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com,dev@example.com&username_env=SMTP_TEST_USER&password_env=SMTP_TEST_PASSWORD&starttls=required",
		Title:  "Signal",
		Body:   "Buy BTCUSDT",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sender.message.Address != "smtp.example.com:587" ||
		sender.message.Username != "bot@example.com" ||
		sender.message.Password != "smtp-secret" ||
		sender.message.StartTLSMode != "required" ||
		sender.message.Subject != "Signal" ||
		sender.message.Body != "Signal\n\nBuy BTCUSDT" {
		t.Fatalf("unexpected mail message: %#v", sender.message)
	}
	if len(sender.message.To) != 2 || sender.message.To[0] != "ops@example.com" || sender.message.To[1] != "dev@example.com" {
		t.Fatalf("unexpected recipients: %#v", sender.message.To)
	}
}

func TestEmailProviderRedactsPasswordFromSenderErrors(t *testing.T) {
	t.Setenv("SMTP_TEST_USER", "bot@example.com")
	t.Setenv("SMTP_TEST_PASSWORD", "smtp-secret")
	provider := NewEmailProvider(&captureMailSender{err: errors.New("auth failed with smtp-secret")})

	err := provider.Deliver(t.Context(), data.NotificationDelivery{
		Target: "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&username_env=SMTP_TEST_USER&password_env=SMTP_TEST_PASSWORD",
		Title:  "Signal",
	})
	if err == nil {
		t.Fatal("expected email error")
	}
	if strings.Contains(err.Error(), "smtp-secret") {
		t.Fatalf("email error leaked password: %v", err)
	}
}

func TestDefaultProvidersIncludeExternalProviders(t *testing.T) {
	registry := DefaultProviders()
	for _, name := range []string{"email", "feishu", "telegram"} {
		if _, err := registry.Provider(name); err != nil {
			t.Fatalf("provider %s not registered: %v", name, err)
		}
	}
}

type captureMailSender struct {
	message MailMessage
	err     error
}

func (sender *captureMailSender) Send(_ context.Context, message MailMessage) error {
	sender.message = message
	return sender.err
}
