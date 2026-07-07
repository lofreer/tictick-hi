package notification

import (
	"strings"
	"testing"
)

func TestProviderMessageIDFromResponseBody(t *testing.T) {
	result := deliveryResultFromResponseBody(strings.NewReader(`{"ok":true,"result":{"message_id":654321}}`))
	if result.ProviderMessageID != "654321" {
		t.Fatalf("provider message id = %q", result.ProviderMessageID)
	}
}

func TestProviderMessageIDFromResponseBodyBoundsText(t *testing.T) {
	result := deliveryResultFromResponseBody(strings.NewReader(`{"messageId":"  ` + strings.Repeat("m", 300) + `  "}`))
	if len([]rune(result.ProviderMessageID)) != maxProviderMessageIDLength {
		t.Fatalf("provider message id length = %d", len([]rune(result.ProviderMessageID)))
	}
	if strings.Contains(result.ProviderMessageID, " ") {
		t.Fatalf("provider message id was not trimmed: %q", result.ProviderMessageID)
	}
}
