package notification

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

func parseTargetURL(target string, scheme string) (*url.URL, url.Values, error) {
	if strings.TrimSpace(target) == "" {
		return nil, nil, fmt.Errorf("%s target is required", scheme)
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s target: %w", scheme, err)
	}
	if parsed.Scheme != scheme {
		return nil, nil, fmt.Errorf("%s target must use %s scheme", scheme, scheme)
	}
	return parsed, parsed.Query(), nil
}

func requiredParam(values url.Values, name string) (string, error) {
	value := strings.TrimSpace(values.Get(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func optionalEnvReference(values url.Values, name string) (string, error) {
	envName := strings.TrimSpace(values.Get(name))
	if envName == "" {
		return "", nil
	}
	if !validEnvName(envName) {
		return "", fmt.Errorf("%s must reference an environment variable name", name)
	}
	return envName, nil
}

func requiredEnvReference(values url.Values, name string) (string, error) {
	envName, err := optionalEnvReference(values, name)
	if err != nil {
		return "", err
	}
	if envName == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return envName, nil
}

func optionalEnv(values url.Values, name string) (string, string, error) {
	envName := strings.TrimSpace(values.Get(name))
	if envName == "" {
		return "", "", nil
	}
	if !validEnvName(envName) {
		return "", "", fmt.Errorf("%s must reference an environment variable name", name)
	}
	value, exists := os.LookupEnv(envName)
	if !exists || value == "" {
		return "", "", fmt.Errorf("environment variable %s is required", envName)
	}
	return envName, value, nil
}

func requiredEnv(values url.Values, name string) (string, string, error) {
	envName, value, err := optionalEnv(values, name)
	if err != nil {
		return "", "", err
	}
	if envName == "" {
		return "", "", fmt.Errorf("%s is required", name)
	}
	return envName, value, nil
}

func validEnvName(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		validHead := char == '_' || (char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z')
		validBody := validHead || (char >= '0' && char <= '9')
		if (index == 0 && !validHead) || (index > 0 && !validBody) {
			return false
		}
	}
	return true
}

func redactedError(message string, secrets ...string) string {
	redacted := message
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "<redacted>")
	}
	return redacted
}

func limitedResponseMessage(reader io.Reader) string {
	body, _ := io.ReadAll(io.LimitReader(reader, 1024))
	message := strings.TrimSpace(string(body))
	if message == "" {
		return "empty response body"
	}
	return message
}

const (
	maxNotificationTitleLength = 200
	maxNotificationBodyLength  = 4000
	maxNotificationTextLength  = 4096
)

func notificationTitle(title string) string {
	return boundedNotificationText(title, maxNotificationTitleLength)
}

func notificationBody(body string) string {
	return boundedNotificationText(body, maxNotificationBodyLength)
}

func notificationText(title string, body string) string {
	title = notificationTitle(title)
	body = notificationBody(body)
	var text string
	switch {
	case title == "":
		text = body
	case body == "":
		text = title
	default:
		text = title + "\n\n" + body
	}
	return boundedNotificationText(text, maxNotificationTextLength)
}

func boundedNotificationText(value string, limit int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > limit {
		return string(runes[:limit])
	}
	return value
}

func setRequestIDHeader(request *http.Request, requestID string) {
	value := safeRequestIDHeaderValue(requestID)
	if value == "" {
		return
	}
	request.Header.Set(outboundRequestIDHeader, value)
}

func setTraceParentHeader(request *http.Request, traceparent string) {
	value := safeTraceParentHeaderValue(traceparent)
	if value == "" {
		return
	}
	request.Header.Set(outboundTraceParentHeader, value)
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

func splitRecipients(value string) ([]string, error) {
	parts := strings.Split(value, ",")
	recipients := make([]string, 0, len(parts))
	for _, part := range parts {
		recipient := strings.TrimSpace(part)
		if recipient != "" {
			recipients = append(recipients, recipient)
		}
	}
	if len(recipients) == 0 {
		return nil, errors.New("at least one recipient is required")
	}
	return recipients, nil
}
