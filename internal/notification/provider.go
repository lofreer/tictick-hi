package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type Provider interface {
	Deliver(ctx context.Context, delivery data.NotificationDelivery) error
}

type ProviderRegistry struct {
	providers map[string]Provider
}

func DefaultProviders() ProviderRegistry {
	provider := DemoProvider{}
	return ProviderRegistry{providers: map[string]Provider{
		"local":        provider,
		"webhook-demo": provider,
		"email":        NewEmailProvider(nil),
		"feishu":       NewFeishuProvider(nil),
		"telegram":     NewTelegramProvider(nil),
		"webhook":      NewWebhookProvider(nil),
	}}
}

func DemoProviders() ProviderRegistry {
	return DefaultProviders()
}

func (registry ProviderRegistry) Provider(name string) (Provider, error) {
	provider, ok := registry.providers[name]
	if !ok {
		return nil, fmt.Errorf("notification provider %q is not registered", name)
	}
	return provider, nil
}

func ValidateProviderTarget(provider string, target string) error {
	switch provider {
	case "local", "webhook-demo":
		return DemoProvider{}.Deliver(context.Background(), data.NotificationDelivery{Target: target})
	case "webhook":
		return validateWebhookTarget(target)
	case "email":
		_, err := parseEmailTarget(target)
		return err
	case "feishu":
		_, err := parseFeishuTarget(target)
		return err
	case "telegram":
		_, err := parseTelegramTarget(target)
		return err
	default:
		_, err := DefaultProviders().Provider(provider)
		return err
	}
}

type DemoProvider struct{}

func (DemoProvider) Deliver(_ context.Context, delivery data.NotificationDelivery) error {
	if delivery.Target == "" {
		return errors.New("notification target is required")
	}
	if strings.Contains(strings.ToLower(delivery.Target), "fail") {
		return fmt.Errorf("demo provider rejected target %q", delivery.Target)
	}
	return nil
}

type WebhookProvider struct {
	client *http.Client
}

func NewWebhookProvider(client *http.Client) WebhookProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return WebhookProvider{client: client}
}

func (provider WebhookProvider) Deliver(ctx context.Context, delivery data.NotificationDelivery) error {
	if err := validateWebhookTarget(delivery.Target); err != nil {
		return err
	}

	payload, err := json.Marshal(webhookPayload{
		NotificationID: delivery.NotificationID,
		DeliveryID:     delivery.ID,
		TaskID:         delivery.TaskID,
		IntentID:       delivery.IntentID,
		RequestID:      safeRequestIDHeaderValue(delivery.RequestID),
		TraceParent:    safeTraceParentHeaderValue(delivery.TraceParent),
		Channel:        delivery.Channel,
		Title:          delivery.Title,
		Body:           delivery.Body,
		AttemptCount:   delivery.AttemptCount,
		MaxAttempts:    delivery.MaxAttempts,
		CreatedAt:      delivery.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("encode webhook notification: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.Target, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create webhook request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	setRequestIDHeader(request, delivery.RequestID)
	setTraceParentHeader(request, delivery.TraceParent)

	response, err := provider.client.Do(request)
	if err != nil {
		return fmt.Errorf("deliver webhook notification: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusBadRequest {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
	message := strings.TrimSpace(string(body))
	if message == "" {
		message = response.Status
	}
	return fmt.Errorf("webhook notification returned HTTP %d: %s", response.StatusCode, message)
}

type webhookPayload struct {
	NotificationID string    `json:"notificationId"`
	DeliveryID     string    `json:"deliveryId"`
	TaskID         string    `json:"taskId,omitempty"`
	IntentID       string    `json:"intentId,omitempty"`
	RequestID      string    `json:"requestId,omitempty"`
	TraceParent    string    `json:"traceparent,omitempty"`
	Channel        string    `json:"channel"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	AttemptCount   int       `json:"attemptCount"`
	MaxAttempts    int       `json:"maxAttempts"`
	CreatedAt      time.Time `json:"createdAt"`
}

func validateWebhookTarget(target string) error {
	if target == "" {
		return errors.New("webhook target is required")
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return fmt.Errorf("parse webhook target: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("webhook target must use http or https")
	}
	if parsed.Host == "" {
		return errors.New("webhook target host is required")
	}
	return nil
}
