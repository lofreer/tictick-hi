package postgres

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationListMarketInstrumentsSearchesActiveCatalog(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := time.Now().UTC().Format("150405000000000")
	activeSymbol := "ITCAT" + suffix + "USDT"
	inactiveSymbol := "ITCATOLD" + suffix + "USDT"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2)`, activeSymbol, inactiveSymbol)
	})

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_instruments (
			exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
		)
		VALUES
			('binance', $1, 'ITCAT', 'USDT', 'spot', 'active', 0, now()),
			('binance', $2, 'ITCATOLD', 'USDT', 'spot', 'inactive', 0, now())`,
		activeSymbol,
		inactiveSymbol,
	); err != nil {
		t.Fatal(err)
	}

	instruments, err := store.ListMarketInstruments(ctx, data.MarketInstrumentQuery{
		Exchange: "binance",
		Query:    "itcat",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 1 || instruments[0].Symbol != activeSymbol {
		t.Fatalf("instruments = %#v, want only active %s", instruments, activeSymbol)
	}
	if instruments[0].BaseAsset != "ITCAT" || instruments[0].QuoteAsset != "USDT" {
		t.Fatalf("unexpected instrument metadata: %#v", instruments[0])
	}

	allInstruments, err := store.ListMarketInstruments(ctx, data.MarketInstrumentQuery{
		Exchange: "binance",
		Query:    "itcat",
		Limit:    10,
		Status:   "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(allInstruments) != 2 || allInstruments[0].Symbol != activeSymbol || allInstruments[1].Symbol != inactiveSymbol {
		t.Fatalf("all instruments = %#v, want active then inactive", allInstruments)
	}

	inactiveInstruments, err := store.ListMarketInstruments(ctx, data.MarketInstrumentQuery{
		Exchange: "binance",
		Query:    "itcat",
		Limit:    10,
		Status:   "inactive",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(inactiveInstruments) != 1 || inactiveInstruments[0].Symbol != inactiveSymbol {
		t.Fatalf("inactive instruments = %#v, want only inactive %s", inactiveInstruments, inactiveSymbol)
	}
}

func TestIntegrationGetActiveMarketInstrumentRequiresExactActiveSymbol(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := time.Now().UTC().Format("150405000000000")
	activeSymbol := "ITEXACT" + suffix + "USDT"
	inactiveSymbol := "ITEXACTOLD" + suffix + "USDT"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2)`, activeSymbol, inactiveSymbol)
	})

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_instruments (
			exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
		)
		VALUES
			('binance', $1, 'ITEXACT', 'USDT', 'spot', 'active', 0, now()),
			('binance', $2, 'ITEXACTOLD', 'USDT', 'spot', 'inactive', 0, now())`,
		activeSymbol,
		inactiveSymbol,
	); err != nil {
		t.Fatal(err)
	}

	instrument, err := store.GetActiveMarketInstrument(ctx, "binance", activeSymbol)
	if err != nil {
		t.Fatal(err)
	}
	if instrument.Symbol != activeSymbol || instrument.Status != "active" {
		t.Fatalf("instrument = %#v, want active %s", instrument, activeSymbol)
	}

	if _, err := store.GetActiveMarketInstrument(ctx, "binance", inactiveSymbol); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("inactive lookup error = %v, want ErrNotFound", err)
	}
	if _, err := store.GetActiveMarketInstrument(ctx, "binance", "ITEXACT"); !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("partial lookup error = %v, want ErrNotFound", err)
	}
}

func TestIntegrationListMarketInstrumentsUsesSeededPriority(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	instruments, err := store.ListMarketInstruments(ctx, data.MarketInstrumentQuery{
		Exchange: "okx",
		Query:    "usdt",
		Limit:    3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 3 {
		t.Fatalf("instruments = %d, want 3: %#v", len(instruments), instruments)
	}
	expected := []string{"BTC-USDT", "ETH-USDT", "SOL-USDT"}
	for index, symbol := range expected {
		if instruments[index].Symbol != symbol {
			t.Fatalf("instrument %d = %s, want %s; all=%#v", index, instruments[index].Symbol, symbol, instruments)
		}
	}
}

func TestIntegrationReplaceMarketInstrumentsMarksMissingActiveInactive(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := time.Now().UTC().Format("150405000000000")
	oldSymbol := "ITREPL" + suffix + "OLD"
	newSymbol := "ITREPL" + suffix + "USDT"
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2)`, oldSymbol, newSymbol)
	})

	existingActive, err := listAllIntegrationActiveInstruments(ctx, store, "binance")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_instruments (
			exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
		)
		VALUES ('binance', $1, 'ITREPL', 'OLD', 'spot', 'active', 5, now())`,
		oldSymbol,
	); err != nil {
		t.Fatal(err)
	}

	replacement := append(existingActive, data.MarketInstrument{
		Symbol: newSymbol, BaseAsset: "ITREPL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active", SearchPriority: 100,
	})
	result, err := store.ReplaceMarketInstruments(
		ctx,
		"binance",
		replacement,
		time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ActiveCount != len(replacement) || result.InactiveCount != 1 {
		t.Fatalf("unexpected sync result: %#v", result)
	}

	instruments, err := store.ListMarketInstruments(ctx, data.MarketInstrumentQuery{
		Exchange: "binance",
		Query:    "ITREPL",
		Limit:    10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(instruments) != 1 || instruments[0].Symbol != newSymbol {
		t.Fatalf("active instruments = %#v, want only %s", instruments, newSymbol)
	}
}

func TestIntegrationReplaceMarketInstrumentsPausesDataSyncTasksForInactiveMarkets(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	symbol := integrationSymbol("PMI")
	taskID := integrationID("dst")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, taskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol = $1`, symbol)
	})

	existingActive, err := listAllIntegrationActiveInstruments(ctx, store, "binance")
	if err != nil {
		t.Fatal(err)
	}
	insertIntegrationSyncTask(t, ctx, store, taskID, symbol, data.TaskStatusRunning, true, true, "market-status-worker")

	result, err := store.ReplaceMarketInstruments(
		ctx,
		"binance",
		append(existingActive, data.MarketInstrument{
			Symbol:         symbol,
			BaseAsset:      "PMI",
			QuoteAsset:     "USDT",
			InstrumentType: "spot",
			Status:         "inactive",
			SearchPriority: 100,
		}),
		time.Date(2026, 6, 29, 10, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.PausedDataSyncTaskCount != 1 {
		t.Fatalf("paused data sync task count = %d, want 1", result.PausedDataSyncTaskCount)
	}

	var (
		syncEnabled     bool
		realtimeEnabled bool
		status          data.TaskStatus
		lockedBy        sql.NullString
		lockedUntil     sql.NullTime
		heartbeatAt     sql.NullTime
	)
	if err := store.pool.QueryRow(ctx, `
		SELECT sync_enabled, realtime_enabled, status, locked_by, locked_until, heartbeat_at
		  FROM data_sync_tasks
		 WHERE id = $1`,
		taskID,
	).Scan(&syncEnabled, &realtimeEnabled, &status, &lockedBy, &lockedUntil, &heartbeatAt); err != nil {
		t.Fatal(err)
	}
	if syncEnabled || realtimeEnabled || status != data.TaskStatusPaused {
		t.Fatalf("task state = sync:%t realtime:%t status:%s, want disabled paused", syncEnabled, realtimeEnabled, status)
	}
	if lockedBy.Valid || lockedUntil.Valid || heartbeatAt.Valid {
		t.Fatalf("task lease not cleared: lockedBy=%#v lockedUntil=%#v heartbeatAt=%#v", lockedBy, lockedUntil, heartbeatAt)
	}
}

func listAllIntegrationActiveInstruments(ctx context.Context, store *Store, exchange string) ([]data.MarketInstrument, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
		  FROM market_instruments
		 WHERE exchange = $1
		   AND status = 'active'`,
		exchange,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instruments []data.MarketInstrument
	for rows.Next() {
		var instrument data.MarketInstrument
		if err := rows.Scan(
			&instrument.Exchange,
			&instrument.Symbol,
			&instrument.BaseAsset,
			&instrument.QuoteAsset,
			&instrument.InstrumentType,
			&instrument.Status,
			&instrument.SearchPriority,
			&instrument.SyncedAt,
		); err != nil {
			return nil, err
		}
		instruments = append(instruments, instrument)
	}
	return instruments, rows.Err()
}
