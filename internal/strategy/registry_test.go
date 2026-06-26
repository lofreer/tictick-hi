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

	_, err = registry.GetStrategy(t.Context(), "missing")
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("missing strategy error = %v", err)
	}
}
