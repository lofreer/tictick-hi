package postgres

import (
	"context"
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
