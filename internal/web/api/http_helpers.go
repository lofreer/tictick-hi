package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func pathParts(requestPath string) []string {
	trimmed := strings.Trim(requestPath, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func readJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	code, safeMessage := defaultError(status, message)
	writeAPIError(w, status, code, safeMessage)
}

func writeMethodNotAllowed(w http.ResponseWriter, allowedMethods ...string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
	writeError(w, http.StatusMethodNotAllowed, "method not allowed")
}

type apiErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func writeAPIError(w http.ResponseWriter, status int, code apiErrorCode, message string) {
	if !apiErrorCodeKnown(code) {
		status = http.StatusInternalServerError
		code = apiErrorInternal
		message = "internal server error"
	}
	writeJSON(w, status, apiErrorResponse{
		Code:    string(code),
		Message: message,
		Error:   message,
	})
}

func defaultError(status int, message string) (apiErrorCode, string) {
	switch status {
	case http.StatusBadRequest:
		return apiErrorInvalidRequest, message
	case http.StatusUnauthorized:
		return apiErrorUnauthorized, message
	case http.StatusForbidden:
		return apiErrorForbidden, message
	case http.StatusNotFound:
		return apiErrorNotFound, message
	case http.StatusMethodNotAllowed:
		return apiErrorMethodNotAllowed, message
	case http.StatusConflict:
		return apiErrorConflict, message
	case http.StatusTooManyRequests:
		return apiErrorTooManyRequests, message
	}
	if status >= http.StatusInternalServerError {
		return apiErrorInternal, "internal server error"
	}
	return apiErrorRequestFailed, message
}

func writeAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, data.ErrUnauthorized) || errors.Is(err, data.ErrNotFound) {
		writeAPIError(w, http.StatusUnauthorized, apiErrorUnauthorized, "unauthorized")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, data.ErrNotFound) {
		writeAPIError(w, http.StatusNotFound, apiErrorNotFound, "not found")
		return
	}
	if code, ok := data.DomainErrorCode(err); ok {
		switch code {
		case data.ErrorCodeDataSyncRetryRequiresFailed:
			writeAPIError(w, http.StatusConflict, apiErrorDataSyncRetryRequiresFailed, err.Error())
			return
		case data.ErrorCodeDataSyncCommandInvalidState:
			writeAPIError(w, http.StatusConflict, apiErrorDataSyncCommandInvalidState, err.Error())
			return
		}
	}
	if errors.Is(err, data.ErrInvalidState) {
		writeAPIError(w, http.StatusConflict, apiErrorInvalidState, err.Error())
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
