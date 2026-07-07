package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

func TestSystemNotificationChannelAuditMetadataExcludesTarget(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/notifications/channels",
		`{"name":"Ops Webhook","provider":"webhook","target":"https://hooks.example.invalid/ops?token=notification-secret","enabled":true}`,
	)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	event := assertAuditAction(t, repository.auditEvents, "notification_channel.create", "notification_channel", "nc_1")
	if event.Metadata["name"] != "Ops Webhook" ||
		event.Metadata["provider"] != "webhook" ||
		event.Metadata["enabled"] != "true" {
		t.Fatalf("unexpected notification channel audit metadata: %#v", event.Metadata)
	}
	if _, exists := event.Metadata["target"]; exists {
		t.Fatalf("notification channel audit metadata includes target: %#v", event.Metadata)
	}
	for key, value := range event.Metadata {
		if strings.Contains(value, "notification-secret") {
			t.Fatalf("notification channel audit metadata leaked target token in %s=%q", key, value)
		}
	}
}

func TestSystemNotificationChannelCreateStoreFailureAudited(t *testing.T) {
	base := newFakeRepository()
	repository := &notificationChannelFailureRepository{
		fakeRepository: base,
		createErr:      errors.New("create failed"),
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	recorder := serveAuthenticated(
		server,
		auth,
		http.MethodPost,
		"/api/system/notifications/channels",
		`{"name":"Ops Webhook","provider":"webhook","target":"https://hooks.example.invalid/ops?token=notification-secret","enabled":true}`,
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("create failure status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Code != "internal_error" || response.Message != "internal server error" {
		t.Fatalf("unexpected create failure response: %#v", response)
	}

	event := assertAuditAction(t, base.auditEvents, "notification_channel.create", "notification_channel", "")
	if event.Outcome != "failure" ||
		event.Metadata["reason"] != "store_error" ||
		event.Metadata["name"] != "Ops Webhook" ||
		event.Metadata["provider"] != "webhook" ||
		event.Metadata["enabled"] != "true" {
		t.Fatalf("unexpected create failure audit metadata: %#v", event.Metadata)
	}
	if _, exists := event.Metadata["target"]; exists {
		t.Fatalf("notification channel failure audit metadata includes target: %#v", event.Metadata)
	}
}

func TestSystemNotificationChannelStoreFailuresAudited(t *testing.T) {
	base := newFakeRepository()
	repository := &notificationChannelFailureRepository{
		fakeRepository: base,
		updateErr:      data.ErrNotFound,
		deleteErr:      data.ErrNotFound,
		enabledErr:     data.ErrNotFound,
	}
	server := NewServer(repository, "")
	auth := loginTestOperator(t, server)

	updateRecorder := serveAuthenticated(
		server,
		auth,
		http.MethodPut,
		"/api/system/notifications/channels/nc_missing",
		`{"name":"Ops","provider":"local","target":"default","enabled":true}`,
	)
	if updateRecorder.Code != http.StatusNotFound {
		t.Fatalf("update failure status = %d body = %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	updateEvent := assertAuditAction(t, base.auditEvents, "notification_channel.update", "notification_channel", "nc_missing")
	if updateEvent.Outcome != "failure" ||
		updateEvent.Metadata["reason"] != "not_found" ||
		updateEvent.Metadata["name"] != "Ops" {
		t.Fatalf("unexpected update failure audit metadata: %#v", updateEvent.Metadata)
	}

	deleteRecorder := serveAuthenticated(server, auth, http.MethodDelete, "/api/system/notifications/channels/nc_missing", "")
	if deleteRecorder.Code != http.StatusNotFound {
		t.Fatalf("delete failure status = %d body = %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	deleteEvent := assertAuditAction(t, base.auditEvents, "notification_channel.delete", "notification_channel", "nc_missing")
	if deleteEvent.Outcome != "failure" || deleteEvent.Metadata["reason"] != "not_found" {
		t.Fatalf("unexpected delete failure audit metadata: %#v", deleteEvent.Metadata)
	}

	disableRecorder := serveAuthenticated(server, auth, http.MethodPost, "/api/system/notifications/channels/nc_missing/disable", "")
	if disableRecorder.Code != http.StatusNotFound {
		t.Fatalf("disable failure status = %d body = %s", disableRecorder.Code, disableRecorder.Body.String())
	}
	disableEvent := assertAuditAction(t, base.auditEvents, "notification_channel.disable", "notification_channel", "nc_missing")
	if disableEvent.Outcome != "failure" ||
		disableEvent.Metadata["reason"] != "not_found" ||
		disableEvent.Metadata["enabled"] != "false" {
		t.Fatalf("unexpected disable failure audit metadata: %#v", disableEvent.Metadata)
	}
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

type notificationChannelFailureRepository struct {
	*fakeRepository
	createErr  error
	updateErr  error
	deleteErr  error
	enabledErr error
}

func (repository *notificationChannelFailureRepository) CreateNotificationChannel(
	ctx context.Context,
	request data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	if repository.createErr != nil {
		return data.NotificationChannel{}, repository.createErr
	}
	return repository.fakeRepository.CreateNotificationChannel(ctx, request)
}

func (repository *notificationChannelFailureRepository) UpdateNotificationChannel(
	ctx context.Context,
	id string,
	request data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	if repository.updateErr != nil {
		return data.NotificationChannel{}, repository.updateErr
	}
	return repository.fakeRepository.UpdateNotificationChannel(ctx, id, request)
}

func (repository *notificationChannelFailureRepository) DeleteNotificationChannel(
	ctx context.Context,
	id string,
) (data.NotificationChannel, error) {
	if repository.deleteErr != nil {
		return data.NotificationChannel{}, repository.deleteErr
	}
	return repository.fakeRepository.DeleteNotificationChannel(ctx, id)
}

func (repository *notificationChannelFailureRepository) SetNotificationChannelEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.NotificationChannel, error) {
	if repository.enabledErr != nil {
		return data.NotificationChannel{}, repository.enabledErr
	}
	return repository.fakeRepository.SetNotificationChannelEnabled(ctx, id, enabled)
}
