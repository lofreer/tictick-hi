package exchange

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const maxEndpointErrorReasonRunes = 180

type HTTPStatusError struct {
	Code   int
	Status string
}

func (err HTTPStatusError) Error() string {
	return "status " + err.Status
}

func EndpointErrorSummary(baseURL string, err error) string {
	return fmt.Sprintf("%s: %s", endpointHost(baseURL), sanitizeEndpointErrorReason(err))
}

func IsTemporaryEndpointError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}

	var statusErr HTTPStatusError
	if errors.As(err, &statusErr) {
		return statusErr.Code == http.StatusTooManyRequests || statusErr.Code >= 500
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, io.EOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return !errors.Is(urlErr.Err, context.Canceled)
	}

	return false
}

func endpointHost(baseURL string) string {
	parsed, err := url.Parse(baseURL)
	if err == nil && parsed.Host != "" {
		return parsed.Host
	}
	return baseURL
}

func sanitizeEndpointErrorReason(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}

	message := strings.Join(strings.Fields(err.Error()), " ")
	runes := []rune(message)
	if len(runes) <= maxEndpointErrorReasonRunes {
		return message
	}
	return string(runes[:maxEndpointErrorReasonRunes-3]) + "..."
}
