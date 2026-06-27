package api

import (
	"net/http"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) handleSystem(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 && parts[2] == "health" && r.Method == http.MethodGet {
		health, err := server.repository.SystemHealth(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, health)
		return
	}
	if len(parts) == 4 && parts[2] == "notifications" && parts[3] == "channels" {
		server.handleNotificationChannels(w, r)
		return
	}
	if len(parts) == 3 && parts[2] == "notifications" && r.Method == http.MethodGet {
		notifications, err := server.repository.ListNotifications(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, notifications)
		return
	}
	if len(parts) == 3 && parts[2] == "audit-events" && r.Method == http.MethodGet {
		events, err := server.repository.ListAuditEvents(r.Context(), parseAuditLimit(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, events)
		return
	}
	if len(parts) == 5 && parts[2] == "notifications" && parts[4] == "retry" && r.Method == http.MethodPost {
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
		if err := server.recordAuditEvent(r, actor, "notification_channel.create", "notification_channel", channel.ID, "success", map[string]string{
			"name":     channel.Name,
			"provider": channel.Provider,
			"enabled":  boolString(channel.Enabled),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, channel)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleOperators(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		operators, err := server.repository.ListOperators(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, operators)
	case http.MethodPost:
		actor, _, authErr := server.currentOperator(r)
		if authErr != nil {
			writeAuthError(w, authErr)
			return
		}
		var request data.CreateOperator
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateOperator(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		operator, err := server.repository.CreateOperator(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := server.recordAuditEvent(r, actor, "operator.create", "operator", operator.ID, "success", map[string]string{
			"username": operator.Username,
			"enabled":  boolString(operator.Enabled),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, operator)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleOperatorAction(w http.ResponseWriter, r *http.Request, id string, action string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
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
		writeError(w, http.StatusNotFound, "operator action not found")
		return
	}
	operator, err := server.repository.SetOperatorEnabled(r.Context(), id, enabled)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	if err := server.recordAuditEvent(r, actor, "operator."+action, "operator", operator.ID, "success", map[string]string{
		"username": operator.Username,
		"enabled":  boolString(operator.Enabled),
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, operator)
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
