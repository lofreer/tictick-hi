package api

import (
	"crypto/rand"
	"encoding/base64"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]loginAttempt
	limit    int
	window   time.Duration
	lockout  time.Duration
	now      func() time.Time
}

type loginAttempt struct {
	failures     int
	firstFailure time.Time
	lockedUntil  time.Time
}

func newLoginLimiter(limit int, window time.Duration, lockout time.Duration) *loginLimiter {
	return &loginLimiter{
		attempts: map[string]loginAttempt{},
		limit:    limit,
		window:   window,
		lockout:  lockout,
		now:      time.Now,
	}
}

func (limiter *loginLimiter) allow(key string) bool {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	attempt := limiter.attempts[key]
	now := limiter.now().UTC()
	if attempt.lockedUntil.After(now) {
		return false
	}
	if !attempt.firstFailure.IsZero() && now.Sub(attempt.firstFailure) > limiter.window {
		delete(limiter.attempts, key)
	}
	return true
}

func (limiter *loginLimiter) recordFailure(key string) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now().UTC()
	attempt := limiter.attempts[key]
	if attempt.firstFailure.IsZero() || now.Sub(attempt.firstFailure) > limiter.window {
		attempt = loginAttempt{firstFailure: now}
	}
	attempt.failures++
	if attempt.failures >= limiter.limit {
		attempt.lockedUntil = now.Add(limiter.lockout)
	}
	limiter.attempts[key] = attempt
}

func (limiter *loginLimiter) recordSuccess(key string) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	delete(limiter.attempts, key)
}

func (server *Server) loginLimitKey(r *http.Request, username string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		host = r.RemoteAddr
	}
	return strings.ToLower(strings.TrimSpace(username)) + "|" + host
}

func (server *Server) validateCSRF(w http.ResponseWriter, r *http.Request) bool {
	if isSafeMethod(r.Method) {
		return true
	}
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, http.StatusForbidden, "csrf token is required")
		return false
	}
	if r.Header.Get(csrfHeaderName) != cookie.Value {
		writeError(w, http.StatusForbidden, "csrf token is invalid")
		return false
	}
	return true
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func newCSRFToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}
