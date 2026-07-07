package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	requestIDHeaderName     = "X-Request-ID"
	generatedRequestIDBytes = 16
)

type requestIDContextKey struct{}

func withRequestID(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	requestID, err := requestIDFromHeader(r.Header.Get(requestIDHeaderName))
	if err != nil {
		return nil, err
	}
	w.Header().Set(requestIDHeaderName, requestID)
	ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
	return r.WithContext(ctx), nil
}

func RequestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func requestIDFromHeader(value string) (string, error) {
	requestID := strings.TrimSpace(value)
	if isValidRequestID(requestID) {
		return requestID, nil
	}
	return generateRequestID()
}

func generateRequestID() (string, error) {
	data := make([]byte, generatedRequestIDBytes)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func isValidRequestID(value string) bool {
	if len(value) < 8 || len(value) > 128 {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '.', '_', ':', '-':
			continue
		default:
			return false
		}
	}
	return true
}
