package postgres

import (
	"strings"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestNotificationProviderMessageIDNormalizesValue(t *testing.T) {
	messageID := notificationProviderMessageID(data.NotificationDeliveryResult{
		ProviderMessageID: "  " + strings.Repeat("m", 300) + "  ",
	})
	if len([]rune(messageID)) != 256 {
		t.Fatalf("message id length = %d, want 256", len([]rune(messageID)))
	}
	if strings.Contains(messageID, " ") {
		t.Fatalf("message id was not trimmed: %q", messageID)
	}
}
