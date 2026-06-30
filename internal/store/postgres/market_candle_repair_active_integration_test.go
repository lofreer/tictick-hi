package postgres

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationMarketCandleRepairsRequireActiveMarketInstrument(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	marketCases := []struct {
		name    string
		missing bool
	}{
		{name: "inactive"},
		{name: "missing", missing: true},
	}
	for _, marketCase := range marketCases {
		t.Run(marketCase.name, func(t *testing.T) {
			start := time.Date(2026, 6, 27, 9, 45, 0, 0, time.UTC)
			symbol := integrationSymbol("MCRA")
			t.Cleanup(func() {
				cleanupCtx, cleanupCancel := testContext(t)
				defer cleanupCancel()
				_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE symbol = $1`, symbol)
				_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_candles WHERE symbol = $1`, symbol)
				_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
				ensurePositivePriceConstraint(t, cleanupCtx, store)
			})
			upsertIntegrationMarketInstrument(t, ctx, store, "binance", symbol, "active")
			for _, minute := range []int{0, 1, 4} {
				insertIntegrationCandle(t, ctx, store, integrationGapScanCandle(symbol, start, minute))
			}
			invalidOpenTime := start.Add(5 * time.Minute)
			insertLegacyInvalidDataHealthCandle(t, ctx, store, symbol, invalidOpenTime)
			if marketCase.missing {
				if _, err := store.pool.Exec(ctx, `
					DELETE FROM market_instruments
					 WHERE exchange = 'binance'
					   AND symbol = $1`,
					symbol,
				); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err := store.pool.Exec(ctx, `
					UPDATE market_instruments
					   SET status = 'inactive',
					       exchange_status = 'BREAK'
					 WHERE exchange = 'binance'
					   AND symbol = $1`,
					symbol,
				); err != nil {
					t.Fatal(err)
				}
			}

			repairCases := []struct {
				name string
				run  func() error
			}{
				{name: "single gap", run: func() error {
					_, err := store.RepairMarketCandleGap(ctx, data.RepairMarketCandleGapRequest{
						Exchange: "binance",
						Symbol:   symbol,
						Interval: "1m",
						From:     start.Add(2 * time.Minute),
						To:       start.Add(4 * time.Minute),
					})
					return err
				}},
				{name: "batch gaps", run: func() error {
					_, err := store.RepairMarketCandleGaps(ctx, data.RepairMarketCandleGapsRequest{
						Exchange: "binance",
						Symbol:   symbol,
						Interval: "1m",
						Gaps: []data.RepairMarketCandleGapWindow{{
							From: start.Add(2 * time.Minute),
							To:   start.Add(4 * time.Minute),
						}},
					})
					return err
				}},
				{name: "invalid issues", run: func() error {
					_, err := store.RepairMarketCandleInvalidIssues(ctx, data.RepairMarketCandleInvalidIssuesRequest{
						Exchange:  "binance",
						Symbol:    symbol,
						Interval:  "1m",
						OpenTimes: []time.Time{invalidOpenTime},
					})
					return err
				}},
			}
			for _, repairCase := range repairCases {
				t.Run(repairCase.name, func(t *testing.T) {
					err := repairCase.run()
					assertMarketInstrumentNotActive(t, err)
				})
			}

			var taskCount int
			if err := store.pool.QueryRow(ctx, `
				SELECT count(*)::int
				  FROM data_sync_tasks
				 WHERE symbol = $1`,
				symbol,
			).Scan(&taskCount); err != nil {
				t.Fatal(err)
			}
			if taskCount != 0 {
				t.Fatalf("%s market repair created %d tasks, want 0", marketCase.name, taskCount)
			}
		})
	}
}
