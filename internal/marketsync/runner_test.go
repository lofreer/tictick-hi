package marketsync

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerRunOnceReplacesEachExchangeAndContinuesAfterFailure(t *testing.T) {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{}
	runner := NewRunner(repository, map[string]exchange.InstrumentClient{
		"binance": fakeInstrumentClient{instruments: []data.MarketInstrument{
			{Symbol: "SOLUSDT", BaseAsset: "SOL", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
		}},
		"okx": fakeInstrumentClient{err: errors.New("okx EOF")},
	}, Config{})
	runner.now = func() time.Time { return now }

	results := runner.RunOnce(context.Background())

	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	if results[0].Exchange != "binance" || results[0].Err != nil {
		t.Fatalf("unexpected first result: %#v", results[0])
	}
	if results[1].Exchange != "okx" || results[1].Err == nil {
		t.Fatalf("unexpected second result: %#v", results[1])
	}
	if len(repository.calls) != 1 {
		t.Fatalf("replace calls = %d, want 1", len(repository.calls))
	}
	call := repository.calls[0]
	if call.exchange != "binance" || !call.syncedAt.Equal(now) || len(call.instruments) != 1 {
		t.Fatalf("unexpected replace call: %#v", call)
	}
}

func TestRunnerRunSyncsOnStartAndInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &countingInstrumentClient{cancelAfter: 2, cancel: cancel}
	runner := NewRunner(&fakeRepository{}, map[string]exchange.InstrumentClient{"binance": client}, Config{
		Interval:    time.Millisecond,
		SyncOnStart: true,
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if client.calls < 2 {
		t.Fatalf("instrument fetch calls = %d, want at least 2", client.calls)
	}
}

type replaceCall struct {
	exchange    string
	instruments []data.MarketInstrument
	syncedAt    time.Time
}

type fakeRepository struct {
	calls []replaceCall
}

func (repository *fakeRepository) ReplaceMarketInstruments(
	_ context.Context,
	exchange string,
	instruments []data.MarketInstrument,
	syncedAt time.Time,
) (data.MarketInstrumentSyncResult, error) {
	repository.calls = append(repository.calls, replaceCall{
		exchange:    exchange,
		instruments: append([]data.MarketInstrument(nil), instruments...),
		syncedAt:    syncedAt,
	})
	return data.MarketInstrumentSyncResult{
		Exchange:    exchange,
		ActiveCount: len(instruments),
		SyncedAt:    syncedAt,
	}, nil
}

type fakeInstrumentClient struct {
	instruments []data.MarketInstrument
	err         error
}

func (client fakeInstrumentClient) FetchInstruments(context.Context) ([]data.MarketInstrument, error) {
	if client.err != nil {
		return nil, client.err
	}
	return append([]data.MarketInstrument(nil), client.instruments...), nil
}

type countingInstrumentClient struct {
	fakeInstrumentClient
	calls       int
	cancelAfter int
	cancel      context.CancelFunc
}

func (client *countingInstrumentClient) FetchInstruments(ctx context.Context) ([]data.MarketInstrument, error) {
	client.calls++
	if client.calls >= client.cancelAfter {
		client.cancel()
	}
	return client.fakeInstrumentClient.FetchInstruments(ctx)
}
