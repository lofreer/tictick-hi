package data

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

const auditEventCursorVersion = 1

type AuditEvent struct {
	ID              string            `json:"id"`
	ActorOperatorID string            `json:"actorOperatorId,omitempty"`
	ActorUsername   string            `json:"actorUsername,omitempty"`
	Action          string            `json:"action"`
	ResourceType    string            `json:"resourceType"`
	ResourceID      string            `json:"resourceId,omitempty"`
	Outcome         string            `json:"outcome"`
	RequestMethod   string            `json:"requestMethod,omitempty"`
	RequestPath     string            `json:"requestPath,omitempty"`
	RemoteAddr      string            `json:"remoteAddr,omitempty"`
	UserAgent       string            `json:"userAgent,omitempty"`
	Metadata        map[string]string `json:"metadata"`
	PreviousHash    string            `json:"previousHash,omitempty"`
	EventHash       string            `json:"eventHash,omitempty"`
	CreatedAt       time.Time         `json:"createdAt"`
}

type AuditEventPage struct {
	Events     []AuditEvent `json:"events"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

type AuditEventListQuery struct {
	Limit  int
	Cursor *AuditEventCursor
}

type AuditEventCursor struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
}

type CreateAuditEvent struct {
	ActorOperatorID string
	ActorUsername   string
	Action          string
	ResourceType    string
	ResourceID      string
	Outcome         string
	RequestMethod   string
	RequestPath     string
	RemoteAddr      string
	UserAgent       string
	Metadata        map[string]string
}

func NewAuditEventCursor(event AuditEvent) AuditEventCursor {
	return AuditEventCursor{
		Version:   auditEventCursorVersion,
		CreatedAt: event.CreatedAt.UTC(),
		ID:        event.ID,
	}
}

func EncodeAuditEventCursor(cursor AuditEventCursor) (string, error) {
	cursor.Version = auditEventCursorVersion
	cursor.CreatedAt = cursor.CreatedAt.UTC()
	if err := validateAuditEventCursor(cursor); err != nil {
		return "", err
	}
	payload, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshal audit event cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func DecodeAuditEventCursor(value string) (AuditEventCursor, error) {
	if value == "" {
		return AuditEventCursor{}, fmt.Errorf("cursor is required")
	}
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return AuditEventCursor{}, fmt.Errorf("cursor is invalid")
	}
	var cursor AuditEventCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return AuditEventCursor{}, fmt.Errorf("cursor is invalid")
	}
	cursor.CreatedAt = cursor.CreatedAt.UTC()
	if err := validateAuditEventCursor(cursor); err != nil {
		return AuditEventCursor{}, err
	}
	return cursor, nil
}

func validateAuditEventCursor(cursor AuditEventCursor) error {
	if cursor.Version != auditEventCursorVersion {
		return fmt.Errorf("cursor version is unsupported")
	}
	if cursor.CreatedAt.IsZero() || cursor.ID == "" {
		return fmt.Errorf("cursor is invalid")
	}
	return nil
}
