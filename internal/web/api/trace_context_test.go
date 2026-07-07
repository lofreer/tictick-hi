package api

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const validTestTraceparent = "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01"

func TestServerAssignsTraceParent(t *testing.T) {
	server := NewServer(newFakeRepository(), "")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get(traceparentHeaderName); !isValidTraceParent(got) {
		t.Fatalf("invalid traceparent response header: %q", got)
	}
}

func TestServerReusesValidTraceParent(t *testing.T) {
	server := NewServer(newFakeRepository(), "")
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.Header.Set(traceparentHeaderName, strings.ToUpper(validTestTraceparent))
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get(traceparentHeaderName); got != validTestTraceparent {
		t.Fatalf("traceparent response header = %q", got)
	}
}

func TestServerReplacesInvalidTraceParentWithoutEchoingValue(t *testing.T) {
	server := NewServer(newFakeRepository(), "")
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	request.Header.Set(traceparentHeaderName, "stage8_config_secret")
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	got := recorder.Header().Get(traceparentHeaderName)
	if !isValidTraceParent(got) {
		t.Fatalf("invalid generated traceparent: %q", got)
	}
	if strings.Contains(got, "stage8_config_secret") {
		t.Fatalf("response leaked invalid traceparent: %q", got)
	}
}

func TestServerAccessLogIncludesTraceID(t *testing.T) {
	var output bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
	t.Cleanup(func() {
		slog.SetDefault(previousLogger)
	})
	server := NewServer(newFakeRepository(), "")
	request := httptest.NewRequest(http.MethodGet, "/readyz?token=stage8_config_secret", nil)
	request.Header.Set(requestIDHeaderName, "request-id-123")
	request.Header.Set(traceparentHeaderName, validTestTraceparent)
	recorder := httptest.NewRecorder()

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	logged := output.String()
	for _, expected := range []string{
		`"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"`,
		`"request_id":"request-id-123"`,
		`"path":"/readyz"`,
	} {
		if !strings.Contains(logged, expected) {
			t.Fatalf("access log missing %s: %s", expected, logged)
		}
	}
	if strings.Contains(logged, "stage8_config_secret") || strings.Contains(logged, "token=") {
		t.Fatalf("access log leaked query string: %s", logged)
	}
}
