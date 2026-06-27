package api

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSystemNotificationChannelsAcceptExternalProviders(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	providers := map[string]string{
		"email":    "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&username_env=SMTP_USERNAME&password_env=SMTP_PASSWORD",
		"telegram": "telegram://send?chat_id=ops-chat&token_env=TELEGRAM_BOT_TOKEN",
		"feishu":   "feishu://webhook?url_env=FEISHU_WEBHOOK_URL",
	}
	for provider, target := range providers {
		recorder := serveAuthenticated(
			server,
			auth,
			http.MethodPost,
			"/api/system/notifications/channels",
			`{"name":"`+provider+`-ops","provider":"`+provider+`","target":"`+target+`","enabled":true}`,
		)
		if recorder.Code != http.StatusCreated {
			t.Fatalf("%s create status = %d body = %s", provider, recorder.Code, recorder.Body.String())
		}
	}

	listRecorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/notifications/channels", "")
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var channels []struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(listRecorder.Body).Decode(&channels); err != nil {
		t.Fatal(err)
	}
	if len(channels) != len(repository.channels) {
		t.Fatalf("channel count mismatch response=%d repository=%d", len(channels), len(repository.channels))
	}
}
