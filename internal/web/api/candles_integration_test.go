package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationCandlesRouteReportsAggregationGapAcrossBasePageBoundary(t *testing.T) {
	store, pool, ctx := openAPIIntegrationStore(t)
	server := NewServer(store, "")

	symbol := apiIntegrationSymbol("APICPB")
	username := fmt.Sprintf("api-candles-gap-%d", time.Now().UTC().UnixNano())
	password := "secret123A"
	start := time.Date(2026, 6, 27, 6, 0, 0, 0, time.UTC)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		cleanupAPIIntegrationMarket(t, cleanupCtx, pool, symbol, username)
	})

	if _, _, err := store.EnsureOperator(ctx, data.CreateOperator{
		Username: username,
		Password: password,
		Enabled:  true,
	}); err != nil {
		t.Fatal(err)
	}
	auth := loginIntegrationOperator(t, server, username, password)
	upsertAPIIntegrationMarketInstrument(t, ctx, pool, symbol)

	targetLimit := data.MaxCandleLimit/60 + 2
	requiredBaseCandles := targetLimit * 60
	missingBaseIndex := data.MaxCandleLimit
	if _, err := pool.Exec(ctx, `
		INSERT INTO market_candles (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, updated_at
		)
		SELECT 'binance',
		       $1,
		       '1m',
		       $2::timestamptz + (series.idx * interval '1 minute'),
		       $2::timestamptz + ((series.idx + 1) * interval '1 minute'),
		       (20000 + series.idx)::numeric,
		       (20001 + series.idx)::numeric,
		       (19999 + series.idx)::numeric,
		       (20000 + series.idx)::numeric,
		       1,
		       true,
		       now()
		  FROM generate_series(0, $3::int - 1) AS series(idx)
		 WHERE series.idx <> $4::int`,
		symbol,
		start,
		requiredBaseCandles,
		missingBaseIndex,
	); err != nil {
		t.Fatal(err)
	}

	path := "/api/candles?exchange=binance&symbol=" + url.QueryEscape(symbol) +
		"&interval=1h&from=" + url.QueryEscape(start.Format(time.RFC3339)) +
		"&limit=" + url.QueryEscape(fmt.Sprintf("%d", targetLimit))
	recorder := serveAuthenticated(server, auth, http.MethodGet, path, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("candles status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	var result data.CandleResult
	if err := json.NewDecoder(recorder.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if result.Source != data.CandleSourceAggregated ||
		result.BaseInterval != "1m" ||
		result.Health != data.CandleHealthGap {
		t.Fatalf("unexpected API candle metadata: %#v", result)
	}
	if result.Coverage.RequiredBaseCandles != requiredBaseCandles ||
		result.Coverage.BaseLimit != requiredBaseCandles ||
		result.Coverage.ReturnedBaseCandles != requiredBaseCandles-1 ||
		result.Coverage.ReturnedCandles != targetLimit-1 ||
		result.Coverage.LimitedByBaseWindow {
		t.Fatalf("unexpected API candle coverage: %#v", result.Coverage)
	}
	if len(result.Gaps) != 1 {
		t.Fatalf("expected one API candle gap across base page boundary: %#v", result.Gaps)
	}
	missingOpen := start.Add(time.Duration(missingBaseIndex) * time.Minute)
	if !result.Gaps[0].From.Equal(missingOpen) ||
		!result.Gaps[0].To.Equal(missingOpen.Add(time.Minute)) ||
		result.Gaps[0].MissingCandles != 1 {
		t.Fatalf("unexpected API candle gap: %#v", result.Gaps[0])
	}

	gappedWindowOpen := missingOpen.Truncate(time.Hour)
	for _, candle := range result.Candles {
		if candle.OpenTime.Equal(gappedWindowOpen) {
			t.Fatalf("API returned aggregated candle for gapped window: %#v", candle)
		}
	}
}
