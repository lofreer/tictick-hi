package data

import (
	"testing"
	"time"
)

func TestAuditEventCursorRoundTrip(t *testing.T) {
	createdAt := time.Date(2026, 1, 2, 3, 4, 5, 6000, time.FixedZone("test", 8*60*60))
	token, err := EncodeAuditEventCursor(NewAuditEventCursor(AuditEvent{
		ID:        "ae_1",
		CreatedAt: createdAt,
	}))
	if err != nil {
		t.Fatal(err)
	}

	cursor, err := DecodeAuditEventCursor(token)
	if err != nil {
		t.Fatal(err)
	}
	if cursor.ID != "ae_1" || !cursor.CreatedAt.Equal(createdAt.UTC()) {
		t.Fatalf("cursor = %#v", cursor)
	}
}

func TestAuditEventCursorRejectsInvalidToken(t *testing.T) {
	if _, err := DecodeAuditEventCursor("not base64"); err == nil {
		t.Fatal("expected invalid cursor to fail")
	}
	if _, err := EncodeAuditEventCursor(AuditEventCursor{ID: "ae_1"}); err == nil {
		t.Fatal("expected missing timestamp to fail")
	}
}
