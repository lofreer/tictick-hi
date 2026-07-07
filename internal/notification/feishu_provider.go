package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type FeishuProvider struct {
	client *http.Client
}

func NewFeishuProvider(client *http.Client) FeishuProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return FeishuProvider{client: client}
}

func (provider FeishuProvider) Deliver(ctx context.Context, delivery data.NotificationDelivery) (data.NotificationDeliveryResult, error) {
	webhookURL, err := parseFeishuTarget(delivery.Target)
	if err != nil {
		return data.NotificationDeliveryResult{}, err
	}
	payload, err := json.Marshal(feishuPayload{
		MessageType: "text",
		Content: feishuTextContent{
			Text: notificationText(delivery.Title, delivery.Body),
		},
	})
	if err != nil {
		return data.NotificationDeliveryResult{}, fmt.Errorf("encode feishu notification: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		return data.NotificationDeliveryResult{}, fmt.Errorf("create feishu request: %s", redactedError(err.Error(), webhookURL))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	setRequestIDHeader(request, delivery.RequestID)
	setTraceParentHeader(request, delivery.TraceParent)

	response, err := provider.client.Do(request)
	if err != nil {
		return data.NotificationDeliveryResult{}, fmt.Errorf("deliver feishu notification: %s", redactedError(err.Error(), webhookURL))
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusBadRequest {
		return deliveryResultFromResponseBody(response.Body), nil
	}
	message := redactedError(limitedResponseMessage(response.Body), webhookURL)
	return data.NotificationDeliveryResult{}, fmt.Errorf("feishu notification returned HTTP %d: %s", response.StatusCode, message)
}

type feishuPayload struct {
	MessageType string            `json:"msg_type"`
	Content     feishuTextContent `json:"content"`
}

type feishuTextContent struct {
	Text string `json:"text"`
}

func parseFeishuTarget(target string) (string, error) {
	_, values, err := parseTargetURL(target, "feishu")
	if err != nil {
		return "", err
	}
	_, webhookURL, err := requiredEnv(values, "url_env")
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(webhookURL, "http://") && !strings.HasPrefix(webhookURL, "https://") {
		return "", fmt.Errorf("feishu webhook url from url_env must use http or https")
	}
	return webhookURL, nil
}

func validateFeishuTargetSyntax(target string) error {
	_, values, err := parseTargetURL(target, "feishu")
	if err != nil {
		return err
	}
	if _, err := requiredEnvReference(values, "url_env"); err != nil {
		return err
	}
	return nil
}
