package notification

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

const maxProviderMessageIDLength = 256

func deliveryResultFromResponseBody(body io.Reader) data.NotificationDeliveryResult {
	messageID := providerMessageIDFromResponseBody(body)
	if messageID == "" {
		return data.NotificationDeliveryResult{}
	}
	return data.NotificationDeliveryResult{ProviderMessageID: messageID}
}

func providerMessageIDFromResponseBody(body io.Reader) string {
	if body == nil {
		return ""
	}
	decoder := json.NewDecoder(io.LimitReader(body, 4096))
	decoder.UseNumber()
	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return ""
	}
	return normalizeProviderMessageID(providerMessageIDFromValue(payload))
}

func providerMessageIDFromValue(value any) string {
	object, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"messageId", "message_id", "id"} {
		if text := providerMessageIDText(object[key]); text != "" {
			return text
		}
	}
	for _, key := range []string{"result", "data"} {
		if text := providerMessageIDFromValue(object[key]); text != "" {
			return text
		}
	}
	return ""
}

func providerMessageIDText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case json.Number:
		return typed.String()
	case float64:
		return fmt.Sprintf("%.0f", typed)
	default:
		return ""
	}
}

func normalizeProviderMessageID(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	runes := []rune(trimmed)
	if len(runes) > maxProviderMessageIDLength {
		runes = runes[:maxProviderMessageIDLength]
	}
	return string(runes)
}
