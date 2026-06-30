package exchange

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const maxEndpointErrorReasonRunes = 180

type HTTPStatusError struct {
	Code            int
	Status          string
	RetryAfterDelay time.Duration
}

func (err HTTPStatusError) Error() string {
	return "status " + err.Status
}

func (err HTTPStatusError) RetryAfter() (time.Duration, bool) {
	return err.RetryAfterDelay, err.RetryAfterDelay > 0
}

func HTTPStatusErrorFromResponse(response *http.Response, now time.Time) HTTPStatusError {
	return HTTPStatusError{
		Code:            response.StatusCode,
		Status:          response.Status,
		RetryAfterDelay: ParseRetryAfterDelay(response.Header.Get("Retry-After"), now),
	}
}

func ParseRetryAfterDelay(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	retryAt, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := retryAt.Sub(now)
	if delay <= 0 {
		return 0
	}
	return delay
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
