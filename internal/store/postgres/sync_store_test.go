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

func TestNormalizeTaskErrorHidesExternalRequestURLs(t *testing.T) {
	err := errors.New(`binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF`)

	normalized := normalizeTaskError(err)
	if normalized != "binance klines: api.binance.com: EOF" {
		t.Fatalf("normalized = %q", normalized)
	}
	for _, forbidden := range []string{`Get "`, "https://", "/api/v3/klines", "symbol=BTCUSDT", "endTime=", "startTime="} {
		if strings.Contains(normalized, forbidden) {
			t.Fatalf("normalized error leaks %q: %s", forbidden, normalized)
		}
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
