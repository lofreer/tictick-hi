package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
		Command:   "sync",
		WorkerID:  "worker-1",
		StartedAt: time.Now().Add(-5 * time.Second),
		ReadinessChecks: []workerReadinessCheck{
			{
				Name: "postgres",
				Check: func(context.Context) error {
					return nil
				},
			},
			{
				Name: "queue",
				Check: func(context.Context) error {
					return nil
				},
			},
		},
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
	var body workerHealthProbeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "ok" ||
		body.Command != "sync" ||
		body.WorkerID != "worker-1" ||
		body.Checks["postgres"] != "ok" ||
		body.Checks["queue"] != "ok" ||
		body.UptimeSeconds < 0 {
		t.Fatalf("unexpected body: %#v", body)
	}
}

func TestWorkerHealthProbeReadinessFailure(t *testing.T) {
	checks := 0
	handler := newWorkerHealthProbeHandler(workerHealthProbeConfig{
		Command:  "trading",
		WorkerID: "worker-2",
		ReadinessChecks: []workerReadinessCheck{
			{
				Name: "postgres",
				Check: func(context.Context) error {
					checks++
					return nil
				},
			},
			{
				Name: "queue",
				Check: func(context.Context) error {
					checks++
					return errors.New("queue unavailable")
				},
			},
		},
	})

	liveRecorder := httptest.NewRecorder()
	handler.ServeHTTP(liveRecorder, httptest.NewRequest(http.MethodGet, "/livez", nil))
	if liveRecorder.Code != http.StatusOK {
		t.Fatalf("livez status = %d, want 200", liveRecorder.Code)
	}
	if checks != 0 {
		t.Fatalf("livez invoked readiness check %d times", checks)
	}

	readyRecorder := httptest.NewRecorder()
	handler.ServeHTTP(readyRecorder, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if readyRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("readyz status = %d, want 503", readyRecorder.Code)
	}
	var body workerHealthProbeResponse
	if err := json.Unmarshal(readyRecorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Status != "unavailable" ||
		body.Checks["postgres"] != "ok" ||
		body.Checks["queue"] != "unavailable" {
		t.Fatalf("unexpected readiness body: %#v", body)
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
