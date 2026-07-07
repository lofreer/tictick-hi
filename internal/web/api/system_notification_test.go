package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
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

func TestSystemNotificationChannelEnableDisable(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.channels = append(repository.channels, dataNotificationChannel("nc_ops", true))

	disableRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/notifications/channels/nc_ops/disable", "")
	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("disable status = %d body = %s", disableRecorder.Code, disableRecorder.Body.String())
	}
	if repository.channels[0].Enabled {
		t.Fatalf("channel enabled = true after disable")
	}

	enableRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/notifications/channels/nc_ops/enable", "")
	if enableRecorder.Code != http.StatusOK {
		t.Fatalf("enable status = %d body = %s", enableRecorder.Code, enableRecorder.Body.String())
	}
	if !repository.channels[0].Enabled {
		t.Fatalf("channel enabled = false after enable")
	}

	assertAuditAction(t, repository.auditEvents, "notification_channel.disable", "notification_channel", "nc_ops")
	assertAuditAction(t, repository.auditEvents, "notification_channel.enable", "notification_channel", "nc_ops")
}

func TestSystemNotificationChannelUpdateDelete(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.channels = append(repository.channels, dataNotificationChannel("nc_ops", true))

	updateRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPut,
		"/api/system/notifications/channels/nc_ops",
		`{"name":"Ops Email","provider":"email","target":"smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com","enabled":false}`,
	)
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("update status = %d body = %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	if repository.channels[0].Name != "Ops Email" ||
		repository.channels[0].Provider != "email" ||
		repository.channels[0].Enabled {
		t.Fatalf("channel after update = %#v", repository.channels[0])
	}

	deleteRecorder := serveAuthenticated(server, auth, http.MethodDelete, "/api/system/notifications/channels/nc_ops", "")
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body = %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	if len(repository.channels) != 0 {
		t.Fatalf("channels after delete = %#v", repository.channels)
	}

	secondDeleteRecorder := serveAuthenticated(server, auth, http.MethodDelete, "/api/system/notifications/channels/nc_ops", "")
	if secondDeleteRecorder.Code != http.StatusNotFound {
		t.Fatalf("second delete status = %d body = %s", secondDeleteRecorder.Code, secondDeleteRecorder.Body.String())
	}

	assertAuditAction(t, repository.auditEvents, "notification_channel.update", "notification_channel", "nc_ops")
	assertAuditAction(t, repository.auditEvents, "notification_channel.delete", "notification_channel", "nc_ops")
}

func TestSystemNotificationChannelUpdateValidatesRequest(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	repository.channels = append(repository.channels, dataNotificationChannel("nc_ops", true))

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPut,
		"/api/system/notifications/channels/nc_ops",
		`{"name":"   ","provider":"local","target":"default","enabled":true}`,
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestSystemNotificationChannelRejectsInvalidProviderTarget(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	createRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/notifications/channels",
		`{"name":"Ops","provider":"webhook","target":"ftp://example.invalid/ops","enabled":true}`,
	)
	if createRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid create status = %d body = %s", createRecorder.Code, createRecorder.Body.String())
	}
	if len(repository.channels) != 0 {
		t.Fatalf("channels after invalid create = %#v", repository.channels)
	}

	repository.channels = append(repository.channels, dataNotificationChannel("nc_ops", true))
	updateRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPut,
		"/api/system/notifications/channels/nc_ops",
		`{"name":"Ops","provider":"telegram","target":"telegram://send?chat_id=ops","enabled":true}`,
	)
	if updateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d body = %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	if repository.channels[0].Provider != "local" {
		t.Fatalf("channel provider changed after invalid update: %#v", repository.channels[0])
	}
}

func TestSystemNotificationChannelActionNotFound(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/notifications/channels/missing/disable", "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("missing channel status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func dataNotificationChannel(id string, enabled bool) data.NotificationChannel {
	return data.NotificationChannel{
		ID:        id,
		Name:      "Ops",
		Provider:  "local",
		Target:    "default",
		Enabled:   enabled,
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}
