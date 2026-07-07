package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type loginLimiter struct {
	limit   int
	window  time.Duration
	lockout time.Duration
	now     func() time.Time
}

const sessionContextMaxLength = 255

func newLoginLimiter(limit int, window time.Duration, lockout time.Duration) *loginLimiter {
	return &loginLimiter{
		limit:   limit,
		window:  window,
		lockout: lockout,
		now:     time.Now,
	}
}

func (limiter *loginLimiter) allow(ctx context.Context, repository data.Repository, key string) (bool, error) {
	return repository.CheckLoginRateLimit(ctx, loginLimitKeyHash(key), limiter.now().UTC(), limiter.window)
}

func (limiter *loginLimiter) recordFailure(ctx context.Context, repository data.Repository, key string) error {
	return repository.RecordLoginFailure(
		ctx,
		loginLimitKeyHash(key),
		limiter.now().UTC(),
		limiter.limit,
		limiter.window,
		limiter.lockout,
	)
}

func (limiter *loginLimiter) recordSuccess(ctx context.Context, repository data.Repository, key string) error {
	return repository.ClearLoginRateLimit(ctx, loginLimitKeyHash(key))
}

func (server *Server) loginLimitKey(r *http.Request, username string) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil || host == "" {
		host = r.RemoteAddr
	}
	return strings.ToLower(strings.TrimSpace(username)) + "|" + host
}

func loginLimitKeyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func sessionContextValue(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > sessionContextMaxLength {
		return string(runes[:sessionContextMaxLength])
	}
	return value
}

func (server *Server) validateCSRF(w http.ResponseWriter, r *http.Request) bool {
	if isSafeMethod(r.Method) {
		return true
	}
	cookie, err := r.Cookie(csrfCookieName)
	if err != nil || cookie.Value == "" {
		writeAPIError(w, http.StatusForbidden, apiErrorCSRFRequired, "csrf token is required")
		return false
	}
	if r.Header.Get(csrfHeaderName) != cookie.Value {
		writeAPIError(w, http.StatusForbidden, apiErrorCSRFInvalid, "csrf token is invalid")
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
