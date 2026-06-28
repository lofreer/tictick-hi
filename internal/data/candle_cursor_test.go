package data

import (
	"testing"
	"time"
)

func TestCandleCursorRoundTrip(t *testing.T) {
	from := time.Date(2026, 1, 1, 0, 0, 0, 123, time.UTC)
	to := from.Add(99 * time.Minute)
	query := CandleQuery{
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		Limit:    100,
	}

	token, err := EncodeCandleCursor(NewCandleCursor(query, from, to, query.Limit))
	if err != nil {
		t.Fatal(err)
	}
	cursor, err := DecodeCandleCursor(token)
	if err != nil {
		t.Fatal(err)
	}

	if !cursor.MatchesQuery(query) {
		t.Fatalf("cursor does not match query: %#v", cursor)
	}
	if cursor.Limit != 100 || !cursor.From.Equal(from) || !cursor.To.Equal(to) {
		t.Fatalf("unexpected cursor: %#v", cursor)
	}
}

func TestCandleCursorRejectsInvalidPayloads(t *testing.T) {
	if _, err := DecodeCandleCursor("not-base64"); err == nil {
		t.Fatal("expected invalid base64 cursor to fail")
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := EncodeCandleCursor(CandleCursor{
		Version:  1,
		Exchange: "binance",
		Symbol:   "BTCUSDT",
		Interval: "1m",
		From:     from.Add(time.Minute),
		To:       from,
		Limit:    100,
	}); err == nil {
		t.Fatal("expected inverted cursor window to fail")
	}
}
