package exchange

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestIsTemporaryEndpointError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "too many requests",
			err:  HTTPStatusError{Code: http.StatusTooManyRequests, Status: "429 Too Many Requests"},
			want: true,
		},
		{
			name: "server error",
			err:  HTTPStatusError{Code: http.StatusBadGateway, Status: "502 Bad Gateway"},
			want: true,
		},
		{
			name: "bad request",
			err:  HTTPStatusError{Code: http.StatusBadRequest, Status: "400 Bad Request"},
			want: false,
		},
		{
			name: "transport EOF",
			err:  &url.Error{Op: "Get", URL: "https://example.com/api?secret=1", Err: io.EOF},
			want: true,
		},
		{
			name: "deadline exceeded",
			err:  context.DeadlineExceeded,
			want: true,
		},
		{
			name: "context canceled",
			err:  context.Canceled,
			want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsTemporaryEndpointError(test.err); got != test.want {
				t.Fatalf("IsTemporaryEndpointError() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEndpointErrorSummaryHidesRequestURL(t *testing.T) {
	err := &url.Error{
		Op:  "Get",
		URL: "https://example.com/api/v1/klines?symbol=BTCUSDT&apiKey=secret",
		Err: errors.New("EOF"),
	}

	summary := EndpointErrorSummary("https://example.com", err)
	if strings.Contains(summary, "/api/v1/klines") ||
		strings.Contains(summary, "symbol=BTCUSDT") ||
		strings.Contains(summary, "apiKey=secret") {
		t.Fatalf("summary leaks request URL: %s", summary)
	}
	if summary != "example.com: EOF" {
		t.Fatalf("summary = %q", summary)
	}
}

func TestEndpointErrorSummaryTruncatesLongReason(t *testing.T) {
	summary := EndpointErrorSummary("https://example.com", errors.New(strings.Repeat("x", 300)))

	const prefix = "example.com: "
	if !strings.HasPrefix(summary, prefix) {
		t.Fatalf("summary missing host: %q", summary)
	}
	reason := strings.TrimPrefix(summary, prefix)
	if len([]rune(reason)) != maxEndpointErrorReasonRunes {
		t.Fatalf("reason length = %d, want %d", len([]rune(reason)), maxEndpointErrorReasonRunes)
	}
	if !strings.HasSuffix(reason, "...") {
		t.Fatalf("truncated reason should end with ellipsis: %q", reason)
	}
}

func TestParseRetryAfterDelay(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	retryAt := now.Add(45 * time.Second).Format(http.TimeFormat)

	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "seconds", value: "7", want: 7 * time.Second},
		{name: "http date", value: retryAt, want: 45 * time.Second},
		{name: "zero", value: "0", want: 0},
		{name: "past date", value: now.Add(-time.Second).Format(http.TimeFormat), want: 0},
		{name: "invalid", value: "soon", want: 0},
		{name: "blank", value: "", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ParseRetryAfterDelay(test.value, now); got != test.want {
				t.Fatalf("ParseRetryAfterDelay(%q) = %s, want %s", test.value, got, test.want)
			}
		})
	}
}

func TestHTTPStatusErrorFromResponseCapturesRetryAfter(t *testing.T) {
	response := &http.Response{
		StatusCode: http.StatusTooManyRequests,
		Status:     "429 Too Many Requests",
		Header:     http.Header{"Retry-After": []string{"11"}},
	}
	err := HTTPStatusErrorFromResponse(response, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	retryAfter, ok := RetryAfter(err)
	if !ok || retryAfter != 11*time.Second {
		t.Fatalf("RetryAfter = %s, %t; want 11s, true", retryAfter, ok)
	}
}
