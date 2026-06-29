package errtext

import (
	"strings"
	"testing"
)

func TestExternalErrorHidesRequestURLs(t *testing.T) {
	raw := `binance klines: Get "https://api.binance.com/api/v3/klines?endTime=1782524388943&interval=1m&limit=500&startTime=1780277926000&symbol=BTCUSDT": EOF`

	sanitized := ExternalError(raw)

	if sanitized != "binance klines: api.binance.com: EOF" {
		t.Fatalf("sanitized = %q", sanitized)
	}
	assertNoExternalURLLeak(t, sanitized)
}

func TestExternalErrorHidesBareURLs(t *testing.T) {
	raw := `okx candles unavailable: https://www.okx.com/api/v5/market/history-candles?instId=BTC-USDT&limit=100 returned EOF`

	sanitized := ExternalError(raw)

	if sanitized != "okx candles unavailable: www.okx.com returned EOF" {
		t.Fatalf("sanitized = %q", sanitized)
	}
	assertNoExternalURLLeak(t, sanitized)
}

func TestExternalErrorNormalizesWhitespaceAndTruncates(t *testing.T) {
	sanitized := ExternalError("temporary\n\n" + strings.Repeat("x", 700))

	if strings.Contains(sanitized, "\n") || strings.Contains(sanitized, "  ") {
		t.Fatalf("sanitized whitespace = %q", sanitized)
	}
	if len([]rune(sanitized)) != maxExternalErrorRunes {
		t.Fatalf("sanitized length = %d, want %d", len([]rune(sanitized)), maxExternalErrorRunes)
	}
	if !strings.HasSuffix(sanitized, "...") {
		t.Fatalf("sanitized should end with ellipsis: %q", sanitized)
	}
}

func assertNoExternalURLLeak(t *testing.T, value string) {
	t.Helper()
	for _, forbidden := range []string{"https://", `Get "`, "/api/", "symbol=BTCUSDT", "endTime=", "startTime=", "instId=BTC-USDT"} {
		if strings.Contains(value, forbidden) {
			t.Fatalf("sanitized error leaks %q: %s", forbidden, value)
		}
	}
}
