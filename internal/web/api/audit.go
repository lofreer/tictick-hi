package api

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	defaultAuditEventLimit = 100
	maxAuditEventLimit     = 500
	auditEventCSVFilename  = "tictick-hi-audit-events.csv"
)

var auditEventCSVHeader = []string{
	"id",
	"createdAt",
	"actorOperatorId",
	"actorUsername",
	"action",
	"resourceType",
	"resourceId",
	"outcome",
	"requestMethod",
	"requestPath",
	"remoteAddr",
	"userAgent",
	"previousHash",
	"eventHash",
	"metadata",
}

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
		RemoteAddr:      sessionContextValue(clientAddress(r)),
		UserAgent:       sessionContextValue(r.UserAgent()),
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
		RemoteAddr:    sessionContextValue(clientAddress(r)),
		UserAgent:     sessionContextValue(r.UserAgent()),
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
	if limit <= 0 {
		return defaultAuditEventLimit
	}
	if limit > maxAuditEventLimit {
		return maxAuditEventLimit
	}
	return limit
}

func parseAuditEventListQuery(r *http.Request) (data.AuditEventListQuery, error) {
	query := data.AuditEventListQuery{Limit: parseAuditLimit(r)}
	value := strings.TrimSpace(r.URL.Query().Get("cursor"))
	if value == "" {
		return query, nil
	}
	cursor, err := data.DecodeAuditEventCursor(value)
	if err != nil {
		return data.AuditEventListQuery{}, err
	}
	query.Cursor = &cursor
	return query, nil
}

func auditEventsCSV(events []data.AuditEvent) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	if err := writer.Write(auditEventCSVHeader); err != nil {
		return nil, err
	}
	for _, event := range events {
		record, err := auditEventCSVRecord(event)
		if err != nil {
			return nil, err
		}
		if err := writer.Write(record); err != nil {
			return nil, err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func auditEventCSVRecord(event data.AuditEvent) ([]string, error) {
	metadata := event.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	return []string{
		auditCSVCell(event.ID),
		event.CreatedAt.UTC().Format(time.RFC3339Nano),
		auditCSVCell(event.ActorOperatorID),
		auditCSVCell(event.ActorUsername),
		auditCSVCell(event.Action),
		auditCSVCell(event.ResourceType),
		auditCSVCell(event.ResourceID),
		auditCSVCell(event.Outcome),
		auditCSVCell(event.RequestMethod),
		auditCSVCell(event.RequestPath),
		auditCSVCell(event.RemoteAddr),
		auditCSVCell(event.UserAgent),
		auditCSVCell(event.PreviousHash),
		auditCSVCell(event.EventHash),
		auditCSVCell(string(metadataJSON)),
	}, nil
}

func auditCSVCell(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

func writeAuditEventsCSV(w http.ResponseWriter, payload []byte) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+auditEventCSVFilename+`"`)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
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
