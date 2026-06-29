package datasync

import (
	"context"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestRunnerCompletesBoundedOneShotTaskOnEmptyFetch(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)
	repository := &fakeSyncRepository{
		task: data.DataSyncTask{
			ID:          "dst_empty",
			Exchange:    "binance",
			Symbol:      "BTCUSDT",
			Interval:    "1m",
			StartTime:   &start,
			EndTime:     &end,
			SyncEnabled: true,
			Status:      data.TaskStatusRunning,
		},
		claimed: true,
	}
	fetcher := &fakeMarketClient{candles: []data.Candle{}}
	runner := NewRunner(repository, exchange.NewRegistry(map[string]exchange.MarketDataClient{
		"binance": fetcher,
	}), Config{WorkerID: "test", BatchLimit: 10})
	runner.now = func() time.Time { return end }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if fetcher.calls != 1 {
		t.Fatalf("fetch calls = %d, want 1", fetcher.calls)
	}
	if repository.saved.TaskID != "dst_empty" || !repository.saved.Completed ||
		repository.saved.LastOpenTime != nil || len(repository.saved.Candles) != 0 {
		t.Fatalf("unexpected empty batch result: %#v", repository.saved)
	}
	if repository.failed != nil || repository.retry != nil {
		t.Fatalf("empty bounded fetch should save terminal result, failed=%v retry=%v", repository.failed, repository.retry)
	}
}
