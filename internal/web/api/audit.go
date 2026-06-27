package api

import (
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

const defaultAuditEventLimit = 100

func (server *Server) recordAuditEvent(
	r *http.Request,
	actor data.Operator,
	action string,
	resourceType string,
	resourceID string,
	outcome string,
	metadata map[string]string,
) error {
	_, err := server.repository.RecordAuditEvent(r.Context(), data.CreateAuditEvent{
		ActorOperatorID: actor.ID,
		ActorUsername:   actor.Username,
		Action:          action,
		ResourceType:    resourceType,
		ResourceID:      resourceID,
		Outcome:         outcome,
		RequestMethod:   r.Method,
		RequestPath:     r.URL.Path,
		RemoteAddr:      clientAddress(r),
		UserAgent:       r.UserAgent(),
		Metadata:        metadata,
	})
	return err
}

func (server *Server) recordAnonymousAuditEvent(
	r *http.Request,
	action string,
	resourceType string,
	resourceID string,
	outcome string,
	metadata map[string]string,
) error {
	_, err := server.repository.RecordAuditEvent(r.Context(), data.CreateAuditEvent{
		Action:        action,
		ResourceType:  resourceType,
		ResourceID:    resourceID,
		Outcome:       outcome,
		RequestMethod: r.Method,
		RequestPath:   r.URL.Path,
		RemoteAddr:    clientAddress(r),
		UserAgent:     r.UserAgent(),
		Metadata:      metadata,
	})
	return err
}

func parseAuditLimit(r *http.Request) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return defaultAuditEventLimit
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return defaultAuditEventLimit
	}
	return limit
}

func clientAddress(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
