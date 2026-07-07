package datasync

import (
	"context"
	"log/slog"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
	"github.com/lofreer/tictick-hi/internal/workerlog"
)

func (runner *Runner) fetchCandles(
	ctx context.Context,
	task data.DataSyncTask,
	client exchange.MarketDataClient,
	request exchange.CandleRequest,
) ([]data.Candle, error) {
	attempts := runner.config.FetchRetries + 1
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		candles, err := client.FetchCandles(ctx, request)
		if err == nil {
			return candles, nil
		}
		lastErr = err
		if !exchange.IsTemporaryError(err) || attempt == attempts || hasRetryAfter(err) {
			return nil, err
		}

		slog.Warn(
			"temporary market data fetch failed; retrying",
			workerlog.TaskTraceAttrs(
				task.ID,
				task.RequestID,
				task.TraceParent,
				"exchange",
				request.Exchange,
				"symbol",
				request.Symbol,
				"interval",
				request.Interval,
				"attempt",
				attempt,
				"max_attempts",
				attempts,
				"error",
				err,
			)...,
		)
		if err := waitForRetry(ctx, runner.config.RetryDelay*time.Duration(attempt)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func hasRetryAfter(err error) bool {
	retryAfter, ok := exchange.RetryAfter(err)
	return ok && retryAfter > 0
}

func waitForRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
