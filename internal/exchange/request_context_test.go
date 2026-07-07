package exchange

import (
	"net/http"
	"testing"
)

func TestApplyRequestMetadataHeaders(t *testing.T) {
	ctx := ContextWithRequestMetadata(
		t.Context(),
		"request-id-1",
		"00-4BF92F3577B34DA6A3CE929D0E0E4736-00F067AA0BA902B7-01",
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}

	ApplyRequestMetadataHeaders(request)

	if got := request.Header.Get("X-Request-ID"); got != "request-id-1" {
		t.Fatalf("X-Request-ID = %q, want request-id-1", got)
	}
	wantTraceparent := "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"
	if got := request.Header.Get("traceparent"); got != wantTraceparent {
		t.Fatalf("traceparent = %q, want %s", got, wantTraceparent)
	}
}

func TestApplyRequestMetadataHeadersSkipsUnsafeValues(t *testing.T) {
	ctx := ContextWithRequestMetadata(
		t.Context(),
		"bad\nrequest-id",
		"00-00000000000000000000000000000000-00f067aa0ba902b7-01",
	)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}

	ApplyRequestMetadataHeaders(request)

	if got := request.Header.Get("X-Request-ID"); got != "" {
		t.Fatalf("X-Request-ID = %q, want empty", got)
	}
	if got := request.Header.Get("traceparent"); got != "" {
		t.Fatalf("traceparent = %q, want empty", got)
	}
}
