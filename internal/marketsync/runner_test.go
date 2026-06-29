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
	if len(repository.failures) != 1 || repository.failures[0].exchange != "okx" {
		t.Fatalf("failure records = %#v, want one okx failure", repository.failures)
	}
	call := repository.calls[0]
	if call.exchange != "binance" || !call.syncedAt.Equal(now) || len(call.instruments) != 1 {
		t.Fatalf("unexpected replace call: %#v", call)
	}
}

func TestRunnerRunOnceRetriesTemporaryInstrumentFailure(t *testing.T) {
	now := time.Date(2026, 6, 28, 12, 0, 0, 0, time.UTC)
	repository := &fakeRepository{}
	client := &scriptedInstrumentClient{
		errs: []error{exchange.NewTemporaryError("okx EOF", errors.New("EOF"))},
		instruments: []data.MarketInstrument{
			{Symbol: "BTC-USDT", BaseAsset: "BTC", QuoteAsset: "USDT", InstrumentType: "spot", Status: "active"},
		},
	}
	runner := NewRunner(repository, map[string]exchange.InstrumentClient{"okx": client}, Config{
		FetchRetries: 2,
		RetryDelay:   time.Nanosecond,
	})
	runner.now = func() time.Time { return now }

	results := runner.RunOnce(context.Background())

	if len(results) != 1 || results[0].Err != nil {
		t.Fatalf("unexpected results: %#v", results)
	}
	if client.calls != 2 {
		t.Fatalf("instrument fetch calls = %d, want 2", client.calls)
	}
	if len(repository.calls) != 1 {
		t.Fatalf("replace calls = %d, want 1", len(repository.calls))
	}
	call := repository.calls[0]
	if call.exchange != "okx" || !call.syncedAt.Equal(now) || len(call.instruments) != 1 {
		t.Fatalf("unexpected replace call: %#v", call)
	}
}

func TestRunnerRunOnceDoesNotRetryPermanentInstrumentFailure(t *testing.T) {
	repository := &fakeRepository{}
	client := &scriptedInstrumentClient{errs: []error{errors.New("bad request")}}
	runner := NewRunner(repository, map[string]exchange.InstrumentClient{"binance": client}, Config{
		FetchRetries: 3,
		RetryDelay:   time.Nanosecond,
	})

	results := runner.RunOnce(context.Background())

	if len(results) != 1 || results[0].Err == nil {
		t.Fatalf("unexpected results: %#v", results)
	}
	if client.calls != 1 {
		t.Fatalf("instrument fetch calls = %d, want 1", client.calls)
	}
	if len(repository.calls) != 0 {
		t.Fatalf("replace calls = %d, want 0", len(repository.calls))
	}
	if len(repository.failures) != 1 || repository.failures[0].exchange != "binance" {
		t.Fatalf("failure records = %#v, want one binance failure", repository.failures)
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

type failureCall struct {
	exchange    string
	err         error
	attemptedAt time.Time
}

type fakeRepository struct {
	calls    []replaceCall
	failures []failureCall
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

func (repository *fakeRepository) RecordMarketInstrumentSyncFailure(
	_ context.Context,
	exchange string,
	syncErr error,
	attemptedAt time.Time,
) error {
	repository.failures = append(repository.failures, failureCall{
		exchange:    exchange,
		err:         syncErr,
		attemptedAt: attemptedAt,
	})
	return nil
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

type scriptedInstrumentClient struct {
	errs        []error
	instruments []data.MarketInstrument
	calls       int
}

func (client *scriptedInstrumentClient) FetchInstruments(context.Context) ([]data.MarketInstrument, error) {
	client.calls++
	if len(client.errs) > 0 {
		err := client.errs[0]
		client.errs = client.errs[1:]
		return nil, err
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
