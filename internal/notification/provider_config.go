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

const outboundRequestIDHeader = "X-Request-ID"

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

func optionalEnv(values url.Values, name string) (string, string, error) {
	envName := strings.TrimSpace(values.Get(name))
	if envName == "" {
		return "", "", nil
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

func notificationText(title string, body string) string {
	title = strings.TrimSpace(title)
	body = strings.TrimSpace(body)
	switch {
	case title == "":
		return body
	case body == "":
		return title
	default:
		return title + "\n\n" + body
	}
}

func setRequestIDHeader(request *http.Request, requestID string) {
	value := safeRequestIDHeaderValue(requestID)
	if value == "" {
		return
	}
	request.Header.Set(outboundRequestIDHeader, value)
}

func safeRequestIDHeaderValue(requestID string) string {
	value := strings.TrimSpace(requestID)
	if value == "" || strings.ContainsAny(value, "\r\n") {
		return ""
	}
	return value
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
