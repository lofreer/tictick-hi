package strategy

import (
	"errors"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestBuiltinRegistry(t *testing.T) {
	registry := BuiltinRegistry()
	strategies, err := registry.ListStrategies(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(strategies) < 2 {
		t.Fatalf("strategies len = %d, want at least 2", len(strategies))
	}

	ema, err := registry.GetStrategy(t.Context(), "ema-cross")
	if err != nil {
		t.Fatal(err)
	}
	if ema.Params[0].Key != "fastPeriod" || ema.SupportedIntents[0] != "order" {
		t.Fatalf("unexpected ema definition: %#v", ema)
	}
	if err := ValidateParams(ema, map[string]any{
		"fastPeriod": 12,
		"slowPeriod": 26,
		"orderSize":  0.01,
		"signalMode": "order",
	}); err != nil {
		t.Fatalf("valid params rejected: %v", err)
	}
	if err := ValidateParams(ema, map[string]any{
		"fastPeriod": 1,
		"slowPeriod": 26,
		"orderSize":  0.01,
		"signalMode": "order",
	}); err == nil {
		t.Fatal("expected out-of-range params to be rejected")
	}
	if err := ValidateParams(ema, map[string]any{
		"fastPeriod": 12,
		"slowPeriod": 26,
		"orderSize":  0.01,
		"signalMode": "webhook",
	}); err == nil {
		t.Fatal("expected invalid select option to be rejected")
	}
	if err := ValidateParams(ema, map[string]any{
		"fastPeriod": 12,
		"slowPeriod": 26,
		"orderSize":  0.01,
		"signalMode": "order",
		"unknown":    true,
	}); err == nil {
		t.Fatal("expected unknown param to be rejected")
	}
	normalized, err := NormalizeParams(ema, map[string]any{
		"fastPeriod": 12,
	})
	if err != nil {
		t.Fatalf("normalize params failed: %v", err)
	}
	if normalized["slowPeriod"] != 26 || normalized["orderSize"] != 0.01 || normalized["signalMode"] != "order" {
		t.Fatalf("defaults were not normalized: %#v", normalized)
	}

	_, err = registry.GetStrategy(t.Context(), "missing")
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("missing strategy error = %v", err)
	}
}
