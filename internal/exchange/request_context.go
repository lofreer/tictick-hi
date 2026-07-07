package exchange

import (
	"context"
	"net/http"
	"strings"
)

const (
	outboundRequestIDHeader    = "X-Request-ID"
	outboundTraceParentHeader  = "traceparent"
	traceparentExpectedLength  = 55
	traceparentTraceIDStart    = 3
	traceparentTraceIDEnd      = 35
	traceparentParentIDStart   = 36
	traceparentParentIDEnd     = 52
	traceparentTraceFlagsStart = 53
)

type requestMetadataContextKey struct{}

type requestMetadata struct {
	RequestID   string
	TraceParent string
}

func ContextWithRequestMetadata(ctx context.Context, requestID string, traceparent string) context.Context {
	metadata := requestMetadata{
		RequestID:   safeRequestIDHeaderValue(requestID),
		TraceParent: safeTraceParentHeaderValue(traceparent),
	}
	if metadata.RequestID == "" && metadata.TraceParent == "" {
		return ctx
	}
	return context.WithValue(ctx, requestMetadataContextKey{}, metadata)
}

func ApplyRequestMetadataHeaders(request *http.Request) {
	metadata, _ := request.Context().Value(requestMetadataContextKey{}).(requestMetadata)
	if metadata.RequestID != "" {
		request.Header.Set(outboundRequestIDHeader, metadata.RequestID)
	}
	if metadata.TraceParent != "" {
		request.Header.Set(outboundTraceParentHeader, metadata.TraceParent)
	}
}

func safeRequestIDHeaderValue(requestID string) string {
	value := strings.TrimSpace(requestID)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
}

func safeTraceParentHeaderValue(traceparent string) string {
	value := strings.ToLower(strings.TrimSpace(traceparent))
	if len(value) != traceparentExpectedLength ||
		value[:2] != "00" ||
		value[2] != '-' ||
		value[traceparentTraceIDEnd] != '-' ||
		value[traceparentParentIDEnd] != '-' {
		return ""
	}
	traceID := value[traceparentTraceIDStart:traceparentTraceIDEnd]
	parentID := value[traceparentParentIDStart:traceparentParentIDEnd]
	traceFlags := value[traceparentTraceFlagsStart:]
	if !isLowerHex(traceID) || !isLowerHex(parentID) || !isLowerHex(traceFlags) {
		return ""
	}
	if isAllZero(traceID) || isAllZero(parentID) {
		return ""
	}
	return value
}

func isLowerHex(value string) bool {
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func isAllZero(value string) bool {
	for _, char := range value {
		if char != '0' {
			return false
		}
	}
	return true
}
