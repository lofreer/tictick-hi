package postgres

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationScanMarketCandleGapsReportsPersistedHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCG")
	cleanupMarketCandles(t, store, symbol)
	for _, minute := range []int{0, 1, 3, 6} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    20,
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 4 || scan.Window.From == nil || !scan.Window.From.Equal(start) ||
		scan.Window.To == nil || !scan.Window.To.Equal(start.Add(6*time.Minute)) {
		t.Fatalf("unexpected scan window: %#v", scan.Window)
	}
	if scan.Limited || scan.TotalCount != 2 || scan.ReturnedCount != 2 || len(scan.Gaps) != 2 {
		t.Fatalf("unexpected gap metadata: %#v", scan)
	}
	assertTaskGap(t, scan.Gaps[0], start.Add(2*time.Minute), start.Add(3*time.Minute), 1)
	assertTaskGap(t, scan.Gaps[1], start.Add(4*time.Minute), start.Add(6*time.Minute), 2)
}

func TestIntegrationScanMarketCandleGapsReportsHealthyHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 7, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCH")
	cleanupMarketCandles(t, store, symbol)
	for _, minute := range []int{0, 1, 2} {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if scan.Window.Count != 3 || scan.TotalCount != 0 || scan.ReturnedCount != 0 || len(scan.Gaps) != 0 || scan.Limited {
		t.Fatalf("unexpected healthy scan: %#v", scan)
	}
}

func TestIntegrationScanMarketCandleGapsReportsLimitedTotal(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	start := time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)
	symbol := integrationSymbol("MCL")
	cleanupMarketCandles(t, store, symbol)
	for minute := 0; minute <= 10; minute += 2 {
		insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
	}

	scan, err := store.ScanMarketCandleGaps(ctx, data.MarketCandleGapScanQuery{
		Exchange: "binance",
		Symbol:   symbol,
		Interval: "1m",
		Limit:    2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !scan.Limited || scan.TotalCount != 5 || scan.ReturnedCount != 2 || len(scan.Gaps) != 2 {
		t.Fatalf("unexpected limited scan: %#v", scan)
	}
	assertTaskGap(t, scan.Gaps[0], start.Add(time.Minute), start.Add(2*time.Minute), 1)
	assertTaskGap(t, scan.Gaps[1], start.Add(3*time.Minute), start.Add(4*time.Minute), 1)
}

func cleanupMarketCandles(t *testing.T, store *Store, symbol string) {
	t.Helper()
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
	})
}

func integrationGapScanCandle(symbol string, start time.Time, minute int) data.Candle {
	openTime := start.Add(time.Duration(minute) * time.Minute)
	return data.Candle{
		Exchange:  "binance",
		Symbol:    symbol,
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		Open:      "100",
		High:      "101",
		Low:       "99",
		Close:     "100",
		Volume:    "1",
		IsClosed:  true,
	}
}
