package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationCandleProviderReportsLegacyInvalidCandle(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("IV")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
		ensurePositivePriceConstraint(t, cleanupCtx, store)
	})

	dropPositivePriceConstraint(t, ctx, store)
	start := time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		VALUES ('binance', $1, '1m', $2, $3, 0, 1, 0, 0.5, 0, true, now())`,
		symbol,
		start,
		start.Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	ensurePositivePriceConstraint(t, ctx, store)

	result, err := store.GetCandles(ctx, data.CandleQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Source != data.CandleSourceNative || result.Health != data.CandleHealthInvalid {
		t.Fatalf("unexpected invalid candle result: %#v", result)
	}
	if len(result.Candles) != 0 || len(result.Issues) != 1 ||
		result.Issues[0].OpenTime == nil ||
		!result.Issues[0].OpenTime.Equal(start) {
		t.Fatalf("unexpected invalid candle issues: %#v", result)
	}
}

func dropPositivePriceConstraint(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	if _, err := store.pool.Exec(ctx, `
		ALTER TABLE market_candles
		DROP CONSTRAINT IF EXISTS market_candles_positive_price_values_check`); err != nil {
		t.Fatal(err)
	}
}

func ensurePositivePriceConstraint(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	if _, err := store.pool.Exec(ctx, `
		ALTER TABLE market_candles
		DROP CONSTRAINT IF EXISTS market_candles_positive_price_values_check`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		ALTER TABLE market_candles
		ADD CONSTRAINT market_candles_positive_price_values_check
		CHECK (open > 0 AND high > 0 AND low > 0 AND close > 0 AND volume >= 0)
		NOT VALID`); err != nil {
		t.Fatal(err)
	}
}
