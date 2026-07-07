package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

func (server *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) < 3 || parts[0] != "api" || parts[1] != "auth" {
		writeError(w, http.StatusNotFound, "auth route not found")
		return
	}

	switch {
	case len(parts) == 3 && parts[2] == "login":
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.handleLogin(w, r)
	case len(parts) == 3 && parts[2] == "me":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		server.handleCurrentOperator(w, r)
	case len(parts) == 3 && parts[2] == "logout":
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		server.handleLogout(w, r)
	case len(parts) == 3 && parts[2] == "sessions":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		server.handleListOperatorSessions(w, r)
	case len(parts) == 4 && parts[2] == "sessions":
		if r.Method != http.MethodDelete {
			writeMethodNotAllowed(w, http.MethodDelete)
			return
		}
		server.handleDeleteOperatorSession(w, r, parts[3])
	default:
		writeError(w, http.StatusNotFound, "auth route not found")
	}
}

func (server *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var request data.LoginRequest
	if err := readJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if request.Username == "" || request.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	limitKey := server.loginLimitKey(r, request.Username)
	allowed, err := server.loginLimiter.allow(r.Context(), server.repository, limitKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("check login rate limit: %w", err).Error())
		return
	}
	if !allowed {
		_ = server.recordAnonymousAuditEvent(r, "auth.login", "operator", request.Username, "failure", map[string]string{
			"reason": "rate_limited",
		})
		writeError(w, http.StatusTooManyRequests, "too many login attempts")
		return
	}
	operator, err := server.repository.AuthenticateOperator(
		r.Context(),
		request.Username,
		request.Password,
	)
	if err != nil {
		if limitErr := server.loginLimiter.recordFailure(r.Context(), server.repository, limitKey); limitErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("record login failure: %w", limitErr).Error())
			return
		}
		_ = server.recordAnonymousAuditEvent(r, "auth.login", "operator", request.Username, "failure", map[string]string{
			"reason": "unauthorized",
		})
		writeAuthError(w, err)
		return
	}
	if err := server.loginLimiter.recordSuccess(r.Context(), server.repository, limitKey); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("clear login rate limit: %w", err).Error())
		return
	}

	token, tokenHash, err := newSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sessionID, err := core.NewPrefixedID("os")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	csrfToken, err := newCSRFToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("generate csrf token: %w", err).Error())
		return
	}
	expiresAt := time.Now().UTC().Add(server.sessionTTL)
	err = server.repository.CreateOperatorSession(r.Context(), data.OperatorSession{
		ID:         sessionID,
		OperatorID: operator.ID,
		TokenHash:  tokenHash,
		ExpiresAt:  expiresAt,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := server.recordAuditEvent(r, operator, "auth.login", "operator", operator.ID, "success", map[string]string{
		"sessionId": sessionID,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	server.setSessionCookie(w, token, expiresAt)
	server.setCSRFCookie(w, csrfToken, expiresAt)
	writeJSON(w, http.StatusOK, operator)
}

func (server *Server) handleListOperatorSessions(w http.ResponseWriter, r *http.Request) {
	operator, tokenHash, err := server.currentOperator(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	sessions, err := server.repository.ListOperatorSessions(r.Context(), operator.ID, tokenHash, time.Now().UTC())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sessions)
}

func (server *Server) handleDeleteOperatorSession(w http.ResponseWriter, r *http.Request, sessionID string) {
	if !server.validateCSRF(w, r) {
		return
	}
	operator, tokenHash, err := server.currentOperator(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	if err := server.repository.DeleteOperatorSessionByID(r.Context(), operator.ID, sessionID, tokenHash); err != nil {
		writeStoreError(w, err)
		return
	}
	if err := server.recordAuditEvent(r, operator, "auth.session_revoke", "operator_session", sessionID, "success", nil); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) handleCurrentOperator(w http.ResponseWriter, r *http.Request) {
	operator, _, err := server.currentOperator(r)
	if err != nil {
		writeAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, operator)
}

func (server *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if !server.validateCSRF(w, r) {
		return
	}
	operator, tokenHash, err := server.currentOperator(r)
	if err == nil {
		if err := server.repository.DeleteOperatorSession(r.Context(), tokenHash); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := server.recordAuditEvent(r, operator, "auth.logout", "operator", operator.ID, "success", nil); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}
	server.clearSessionCookie(w)
	server.clearCSRFCookie(w)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) authenticateRequest(w http.ResponseWriter, r *http.Request) (data.Operator, bool) {
	operator, _, err := server.currentOperator(r)
	if err != nil {
		writeAuthError(w, err)
		return data.Operator{}, false
	}
	return operator, true
}

func (server *Server) currentOperator(r *http.Request) (data.Operator, string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return data.Operator{}, "", data.ErrUnauthorized
	}
	tokenHash := sessionTokenHash(cookie.Value)
	operator, err := server.repository.GetOperatorBySession(r.Context(), tokenHash, time.Now().UTC())
	if err != nil {
		return data.Operator{}, "", err
	}
	return operator, tokenHash, nil
}

func (server *Server) setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(server.sessionTTL.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   server.cookieSecure,
	})
}

func (server *Server) setCSRFCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(server.sessionTTL.Seconds()),
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   server.cookieSecure,
	})
}

func (server *Server) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   server.cookieSecure,
	})
}

func (server *Server) clearCSRFCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     csrfCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   server.cookieSecure,
	})
}

func newSessionToken() (string, string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", "", fmt.Errorf("generate session token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes[:])
	return token, sessionTokenHash(token), nil
}

func sessionTokenHash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}
