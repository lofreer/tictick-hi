package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	traceparentHeaderName = "traceparent"
	traceIDBytes          = 16
	traceSpanIDBytes      = 8
)

type traceContextContextKey struct{}

func withTraceContext(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
	traceparent, err := traceparentFromHeader(r.Header.Get(traceparentHeaderName))
	if err != nil {
		return nil, err
	}
	w.Header().Set(traceparentHeaderName, traceparent)
	ctx := context.WithValue(r.Context(), traceContextContextKey{}, traceparent)
	return r.WithContext(ctx), nil
}

func TraceParentFromContext(ctx context.Context) string {
	traceparent, _ := ctx.Value(traceContextContextKey{}).(string)
	return traceparent
}

func TraceIDFromContext(ctx context.Context) string {
	traceparent := TraceParentFromContext(ctx)
	if len(traceparent) != 55 {
		return ""
	}
	return traceparent[3:35]
}

func traceparentFromHeader(value string) (string, error) {
	traceparent := strings.TrimSpace(value)
	if isValidTraceParent(traceparent) {
		return strings.ToLower(traceparent), nil
	}
	return generateTraceParent()
}

func generateTraceParent() (string, error) {
	traceID, err := randomNonZeroHex(traceIDBytes)
	if err != nil {
		return "", err
	}
	spanID, err := randomNonZeroHex(traceSpanIDBytes)
	if err != nil {
		return "", err
	}
	return "00-" + traceID + "-" + spanID + "-00", nil
}

func randomNonZeroHex(size int) (string, error) {
	for {
		data := make([]byte, size)
		if _, err := rand.Read(data); err != nil {
			return "", err
		}
		value := hex.EncodeToString(data)
		if !isAllZeroHex(value) {
			return value, nil
		}
	}
}

func isValidTraceParent(value string) bool {
	if len(value) != 55 {
		return false
	}
	if value[2] != '-' || value[35] != '-' || value[52] != '-' {
		return false
	}
	version := value[0:2]
	traceID := value[3:35]
	spanID := value[36:52]
	flags := value[53:55]
	if strings.EqualFold(version, "ff") || !isHex(version) {
		return false
	}
	if !strings.EqualFold(version, "00") {
		return false
	}
	if !isHex(traceID) || isAllZeroHex(traceID) {
		return false
	}
	if !isHex(spanID) || isAllZeroHex(spanID) {
		return false
	}
	return isHex(flags)
}

func isHex(value string) bool {
	for _, char := range value {
		if char >= '0' && char <= '9' {
			continue
		}
		if char >= 'a' && char <= 'f' {
			continue
		}
		if char >= 'A' && char <= 'F' {
			continue
		}
		return false
	}
	return true
}

func isAllZeroHex(value string) bool {
	for _, char := range value {
		if char != '0' {
			return false
		}
	}
	return true
}
