package api

import (
	"net/http"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 && parts[2] == "health" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		health, err := server.repository.SystemHealth(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, health)
		return
	}
	if len(parts) == 3 && parts[2] == "api-contract" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, apiContractDocument())
		return
	}
	if len(parts) == 4 && parts[2] == "notifications" && parts[3] == "channels" {
		server.handleNotificationChannels(w, r)
		return
	}
	if len(parts) == 5 && parts[2] == "notifications" && parts[3] == "channels" {
		server.handleNotificationChannel(w, r, parts[4])
		return
	}
	if len(parts) == 6 && parts[2] == "notifications" && parts[3] == "channels" {
		server.handleNotificationChannelAction(w, r, parts[4], parts[5])
		return
	}
	if len(parts) == 3 && parts[2] == "notifications" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		notifications, err := server.repository.ListNotifications(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, notifications)
		return
	}
	if len(parts) == 5 && parts[2] == "audit-events" && parts[3] == "hash-chain" && parts[4] == "verify" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		verification, err := server.repository.VerifyAuditEventHashChain(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, verification)
		return
	}
	if len(parts) == 4 && parts[2] == "audit-events" && parts[3] == "page" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		query, err := parseAuditEventListQuery(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		page, err := server.repository.ListAuditEventPage(r.Context(), query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, page)
		return
	}
	if len(parts) == 4 && parts[2] == "audit-events" && parts[3] == "export" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		events, err := server.repository.ListAuditEvents(r.Context(), parseAuditLimit(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		payload, err := auditEventsCSV(events)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeAuditEventsCSV(w, payload)
		return
	}
	if len(parts) == 3 && parts[2] == "audit-events" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		events, err := server.repository.ListAuditEvents(r.Context(), parseAuditLimit(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, events)
		return
	}
	if len(parts) == 5 && parts[2] == "notifications" && parts[4] == "retry" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		notification, err := server.repository.RetryNotification(r.Context(), parts[3])
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := server.recordAuditEvent(r, actor, "notification.retry", "notification", notification.ID, "success", nil); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, notification)
		return
	}
	if len(parts) == 3 && parts[2] == "exchange-accounts" {
		server.handleExchangeAccounts(w, r)
		return
	}
	if len(parts) == 3 && parts[2] == "operators" {
		server.handleOperators(w, r)
		return
	}
	if len(parts) == 5 && parts[2] == "operators" {
		server.handleOperatorAction(w, r, parts[3], parts[4])
		return
	}
	writeError(w, http.StatusNotFound, "system route not found")
}

func (server *Server) handleNotificationChannels(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		channels, err := server.repository.ListNotificationChannels(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, channels)
	case http.MethodPost:
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		var request data.CreateNotificationChannel
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateNotificationChannel(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		channel, err := server.repository.CreateNotificationChannel(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := server.recordAuditEvent(r, actor, "notification_channel.create", "notification_channel", channel.ID, "success", notificationChannelAuditMetadata(channel)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, channel)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func (server *Server) handleNotificationChannel(w http.ResponseWriter, r *http.Request, id string) {
	switch r.Method {
	case http.MethodPut:
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		var request data.CreateNotificationChannel
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateNotificationChannel(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		channel, err := server.repository.UpdateNotificationChannel(r.Context(), id, request)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := server.recordAuditEvent(r, actor, "notification_channel.update", "notification_channel", channel.ID, "success", notificationChannelAuditMetadata(channel)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, channel)
	case http.MethodDelete:
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		channel, err := server.repository.DeleteNotificationChannel(r.Context(), id)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := server.recordAuditEvent(r, actor, "notification_channel.delete", "notification_channel", channel.ID, "success", notificationChannelAuditMetadata(channel)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
	}
}

func (server *Server) handleNotificationChannelAction(w http.ResponseWriter, r *http.Request, id string, action string) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}
	actor, _, authErr := server.currentOperator(r)
	if authErr != nil {
		writeAuthError(w, authErr)
		return
	}
	var enabled bool
	switch action {
	case "enable":
		enabled = true
	case "disable":
		enabled = false
	default:
		writeError(w, http.StatusNotFound, "notification channel action not found")
		return
	}
	channel, err := server.repository.SetNotificationChannelEnabled(r.Context(), id, enabled)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := server.recordAuditEvent(r, actor, "notification_channel."+action, "notification_channel", channel.ID, "success", notificationChannelAuditMetadata(channel)); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, channel)
}

func notificationChannelAuditMetadata(channel data.NotificationChannel) map[string]string {
	return map[string]string{
		"name":     channel.Name,
		"provider": channel.Provider,
		"enabled":  boolString(channel.Enabled),
	}
}

func (server *Server) handleExchangeAccounts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		accounts, err := server.repository.ListExchangeAccounts(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, accounts)
	case http.MethodPost:
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		var request data.CreateExchangeAccount
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateExchangeAccount(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		account, err := server.repository.CreateExchangeAccount(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := server.recordAuditEvent(r, actor, "exchange_account.create", "exchange_account", account.ID, "success", map[string]string{
			"exchange": account.Exchange,
			"alias":    account.Alias,
			"enabled":  boolString(account.Enabled),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, account)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}
