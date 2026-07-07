package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoadWorkerHealthProbeAddrRejectsInvalidAddress(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("SYNC_HEALTH_ADDR", "not-a-host-port")

	_, err := loadWorkerHealthProbeAddr("sync")
	if err == nil || !strings.Contains(err.Error(), "SYNC_HEALTH_ADDR") {
		t.Fatalf("expected SYNC_HEALTH_ADDR error, got %v", err)
	}
}

func TestLoadWorkerHealthProbeAddrAcceptsHostPort(t *testing.T) {
	clearCommandEnv(t)
	t.Setenv("NOTIFY_HEALTH_ADDR", "127.0.0.1:19081")

	addr, err := loadWorkerHealthProbeAddr("notify")
	if err != nil {
		t.Fatalf("load worker health probe addr: %v", err)
	}
	if addr != "127.0.0.1:19081" {
		t.Fatalf("addr = %q, want configured address", addr)
	}
}

func TestWorkerHealthProbeHandler(t *testing.T) {
	handler := newWorkerHealthProbeHandler(workerHealthProbeConfig{
		Command:  "sync",
		WorkerID: "worker-1",
	})
	request := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if recorder.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", recorder.Header().Get("Cache-Control"))
	}
	var body map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" || body["command"] != "sync" || body["workerId"] != "worker-1" {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestWorkerHealthProbeHandlerRejectsWrites(t *testing.T) {
	handler := newWorkerHealthProbeHandler(workerHealthProbeConfig{Command: "notify"})
	request := httptest.NewRequest(http.MethodPost, "/livez", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
	if recorder.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("Allow = %q, want GET, HEAD", recorder.Header().Get("Allow"))
	}
}

func TestStartWorkerHealthProbeServesReadiness(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr, err := startWorkerHealthProbe(ctx, workerHealthProbeConfig{
		Command:  "backtest",
		Addr:     "127.0.0.1:0",
		WorkerID: "worker-probe",
	})
	if err != nil {
		t.Fatalf("start worker health probe: %v", err)
	}

	response, err := http.Get("http://" + addr + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.StatusCode)
	}
}
