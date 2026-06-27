package postgres

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeTaskError(t *testing.T) {
	err := errors.New("binance klines temporary unavailable:\n\napi.binance.com: Get EOF")

	normalized := normalizeTaskError(err)
	if strings.Contains(normalized, "\n") || strings.Contains(normalized, "  ") {
		t.Fatalf("error was not normalized: %q", normalized)
	}
	if normalized != "binance klines temporary unavailable: api.binance.com: Get EOF" {
		t.Fatalf("normalized = %q", normalized)
	}
}

func TestNormalizeTaskErrorTruncatesLongMessages(t *testing.T) {
	normalized := normalizeTaskError(errors.New(strings.Repeat("x", 700)))

	if len([]rune(normalized)) != 500 {
		t.Fatalf("normalized length = %d, want 500", len([]rune(normalized)))
	}
	if !strings.HasSuffix(normalized, "...") {
		t.Fatalf("normalized should end with ellipsis: %q", normalized)
	}
}
