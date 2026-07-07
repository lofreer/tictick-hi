package api

import (
	"bytes"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginRateLimit(t *testing.T) {
	repository := newFakeRepository()
	config := Config{
		LoginFailureLimit:  2,
		LoginFailureWindow: time.Minute,
		LoginLockout:       time.Hour,
	}
	server := NewServerWithConfig(repository, config)

	body := `{"username":"` + testUsername + `","password":"wrong"}`
	for index := 0; index < 2; index++ {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
		request.RemoteAddr = "203.0.113.10:12345"
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status = %d body = %s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	server = NewServerWithConfig(repository, config)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(body))
	request.RemoteAddr = "203.0.113.10:12345"
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginRejectsBlankUsername(t *testing.T) {
	server := NewServer(newFakeRepository(), "")

	recorder := httptestPostJSON(server, "/api/auth/login", `{"username":"   ","password":"secret123A"}`)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("blank username status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	response := decodeAPIError(t, recorder)
	if response.Message != "username and password are required" {
		t.Fatalf("unexpected blank username response: %#v", response)
	}
}

func TestLoginRateLimitClearsAfterSuccessfulLogin(t *testing.T) {
	repository := newFakeRepository()
	server := NewServerWithConfig(repository, Config{
		LoginFailureLimit:  2,
		LoginFailureWindow: time.Minute,
		LoginLockout:       time.Hour,
	})
	remoteAddr := "203.0.113.11:12345"

	wrongBody := `{"username":"` + testUsername + `","password":"wrong"}`
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(wrongBody))
	request.RemoteAddr = remoteAddr
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("first failed status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	successBody := `{"username":"` + testUsername + `","password":"` + testPassword + `"}`
	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(successBody))
	request.RemoteAddr = remoteAddr
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("success status = %d body = %s", recorder.Code, recorder.Body.String())
	}

	for index := 0; index < 2; index++ {
		recorder = httptest.NewRecorder()
		request = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(wrongBody))
		request.RemoteAddr = remoteAddr
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("post-success attempt %d status = %d body = %s", index+1, recorder.Code, recorder.Body.String())
		}
	}

	recorder = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString(wrongBody))
	request.RemoteAddr = remoteAddr
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("limited status = %d body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestLoginLimitKeyHashDoesNotExposeRawKey(t *testing.T) {
	rawKey := "admin|203.0.113.10"

	keyHash := loginLimitKeyHash(rawKey)

	if len(keyHash) != 64 {
		t.Fatalf("hash length = %d", len(keyHash))
	}
	if _, err := hex.DecodeString(keyHash); err != nil {
		t.Fatalf("hash is not hex: %v", err)
	}
	if strings.Contains(keyHash, "admin") || strings.Contains(keyHash, "203.0.113.10") {
		t.Fatalf("hash exposes raw key: %s", keyHash)
	}
}
