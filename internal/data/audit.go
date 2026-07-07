package data

import "time"

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
