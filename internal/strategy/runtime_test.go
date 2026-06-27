package strategy

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestGenerateEMACrossIntents(t *testing.T) {
	definition, err := BuiltinRegistry().GetStrategy(t.Context(), "ema-cross")
	if err != nil {
		t.Fatal(err)
	}

	intents, err := GenerateIntents(t.Context(), definition, testCandles([]string{
		"10", "9", "8", "11", "12", "10", "8",
	}), map[string]any{
		"fastPeriod": 2,
		"slowPeriod": 3,
		"orderSize":  0.5,
		"signalMode": "order",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(intents) == 0 {
		t.Fatal("expected at least one ema intent")
	}
	if intents[0].Quantity != "0.5" {
		t.Fatalf("quantity = %s", intents[0].Quantity)
	}
	if intents[0].Type != IntentTypeOrder || intents[0].Payload["side"] != intents[0].Side {
		t.Fatalf("unexpected order intent payload: %#v", intents[0])
	}
}

func TestGenerateNotificationIntents(t *testing.T) {
	definition, err := BuiltinRegistry().GetStrategy(t.Context(), "ema-cross")
	if err != nil {
		t.Fatal(err)
	}

	intents, err := GenerateIntents(t.Context(), definition, testCandles([]string{
		"10", "9", "8", "11", "12", "10", "8",
	}), map[string]any{
		"fastPeriod": 2,
		"slowPeriod": 3,
		"orderSize":  0.5,
		"signalMode": "notification",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(intents) == 0 {
		t.Fatal("expected at least one notification intent")
	}
	if intents[0].Type != IntentTypeNotification || intents[0].Message == "" {
		t.Fatalf("unexpected notification intent: %#v", intents[0])
	}
}

func TestGenerateBreakoutRangeIntents(t *testing.T) {
	definition, err := BuiltinRegistry().GetStrategy(t.Context(), "breakout-range")
	if err != nil {
		t.Fatal(err)
	}

	intents, err := GenerateIntents(t.Context(), definition, testCandles([]string{
		"10", "10.2", "10.1", "10.3", "10.2", "12",
	}), map[string]any{
		"lookback":          5,
		"breakoutBufferPct": 0.1,
		"orderSize":         0.25,
		"signalMode":        "order",
		"side":              "both",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(intents) != 1 || intents[0].Side != "buy" {
		t.Fatalf("unexpected intents: %#v", intents)
	}
}

func testCandles(closes []string) []data.Candle {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	candles := make([]data.Candle, 0, len(closes))
	for index, closePrice := range closes {
		openTime := start.Add(time.Duration(index) * time.Minute)
		candles = append(candles, data.Candle{
			Exchange:  "binance",
			Symbol:    "BTCUSDT",
			Interval:  "1m",
			OpenTime:  openTime,
			CloseTime: openTime.Add(time.Minute),
			Open:      closePrice,
			High:      closePrice,
			Low:       closePrice,
			Close:     closePrice,
			Volume:    "1",
			IsClosed:  true,
		})
	}
	return candles
}
