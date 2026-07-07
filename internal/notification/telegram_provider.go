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

const defaultTelegramAPIBase = "https://api.telegram.org"

type TelegramProvider struct {
	client *http.Client
}

func NewTelegramProvider(client *http.Client) TelegramProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	return TelegramProvider{client: client}
}

func (provider TelegramProvider) Deliver(ctx context.Context, delivery data.NotificationDelivery) (data.NotificationDeliveryResult, error) {
	config, err := parseTelegramTarget(delivery.Target)
	if err != nil {
		return data.NotificationDeliveryResult{}, err
	}
	payload, err := json.Marshal(telegramPayload{
		ChatID: config.ChatID,
		Text:   notificationText(delivery.Title, delivery.Body),
	})
	if err != nil {
		return data.NotificationDeliveryResult{}, fmt.Errorf("encode telegram notification: %w", err)
	}

	requestURL := strings.TrimRight(config.APIBase, "/") + "/bot" + config.Token + "/sendMessage"
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, requestURL, bytes.NewReader(payload))
	if err != nil {
		return data.NotificationDeliveryResult{}, fmt.Errorf("create telegram request: %s", redactedError(err.Error(), config.Token))
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")
	setRequestIDHeader(request, delivery.RequestID)
	setTraceParentHeader(request, delivery.TraceParent)

	response, err := provider.client.Do(request)
	if err != nil {
		return data.NotificationDeliveryResult{}, fmt.Errorf("deliver telegram notification: %s", redactedError(err.Error(), config.Token))
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusBadRequest {
		return deliveryResultFromResponseBody(response.Body), nil
	}
	message := redactedError(limitedResponseMessage(response.Body), config.Token)
	return data.NotificationDeliveryResult{}, fmt.Errorf("telegram notification returned HTTP %d: %s", response.StatusCode, message)
}

type telegramTarget struct {
	ChatID  string
	Token   string
	APIBase string
}

type telegramPayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func parseTelegramTarget(target string) (telegramTarget, error) {
	_, values, err := parseTargetURL(target, "telegram")
	if err != nil {
		return telegramTarget{}, err
	}
	chatID, err := requiredParam(values, "chat_id")
	if err != nil {
		return telegramTarget{}, err
	}
	_, token, err := requiredEnv(values, "token_env")
	if err != nil {
		return telegramTarget{}, err
	}
	if strings.ContainsAny(token, " \t\r\n/") {
		return telegramTarget{}, fmt.Errorf("telegram token from token_env is invalid")
	}
	apiBase := strings.TrimSpace(values.Get("api_base"))
	if apiBase == "" {
		apiBase = defaultTelegramAPIBase
	}
	if !strings.HasPrefix(apiBase, "http://") && !strings.HasPrefix(apiBase, "https://") {
		return telegramTarget{}, fmt.Errorf("api_base must use http or https")
	}
	return telegramTarget{ChatID: chatID, Token: token, APIBase: apiBase}, nil
}

func validateTelegramTargetSyntax(target string) error {
	_, values, err := parseTargetURL(target, "telegram")
	if err != nil {
		return err
	}
	if _, err := requiredParam(values, "chat_id"); err != nil {
		return err
	}
	if _, err := requiredEnvReference(values, "token_env"); err != nil {
		return err
	}
	apiBase := strings.TrimSpace(values.Get("api_base"))
	if apiBase != "" && !strings.HasPrefix(apiBase, "http://") && !strings.HasPrefix(apiBase, "https://") {
		return fmt.Errorf("api_base must use http or https")
	}
	return nil
}
