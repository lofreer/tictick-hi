package exchange

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
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
