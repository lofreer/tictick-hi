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
	writeJSON(w, status, map[string]string{"error": message})
}

func writeAuthError(w http.ResponseWriter, err error) {
	if errors.Is(err, data.ErrUnauthorized) || errors.Is(err, data.ErrNotFound) {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, data.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
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
