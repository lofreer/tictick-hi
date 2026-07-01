package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) QuarantineMarketCandleInvalidIssues(
	ctx context.Context,
	request data.QuarantineMarketCandleInvalidIssuesRequest,
) (data.MarketCandleQuarantineResult, error) {
	if err := data.ValidateDataSyncTaskWindow(request.Interval, nil, nil); err != nil {
		return data.MarketCandleQuarantineResult{}, err
	}
	duration, err := data.IntervalDuration(request.Interval)
	if err != nil {
		return data.MarketCandleQuarantineResult{}, err
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.MarketCandleQuarantineResult{}, fmt.Errorf("begin quarantine market candle invalid issues: %w", err)
	}
	defer tx.Rollback(ctx)

	result := data.MarketCandleQuarantineResult{
		Quarantined:     []data.MarketCandleQuarantineRecord{},
		TotalCount:      len(request.OpenTimes),
		QuarantineLimit: data.MaxMarketCandleInvalidIssueScanLimit,
	}
	for _, rawOpenTime := range request.OpenTimes {
		row, ok, err := lockMarketCandleInvalidIssue(ctx, tx, request, rawOpenTime.UTC(), duration)
		if err != nil {
			return data.MarketCandleQuarantineResult{}, err
		}
		if !ok {
			return data.MarketCandleQuarantineResult{}, data.ErrNotFound
		}
		if row.code != data.CandleIssueInvalidOpenTime {
			result.SkippedNonQuarantinable++
			continue
		}
		record, err := quarantineMarketCandle(ctx, tx, row)
		if err != nil {
			return data.MarketCandleQuarantineResult{}, err
		}
		result.Quarantined = append(result.Quarantined, record)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.MarketCandleQuarantineResult{}, fmt.Errorf("commit quarantine market candle invalid issues: %w", err)
	}
	return result, nil
}

type marketCandleInvalidIssueRow struct {
	candle  data.Candle
	code    string
	message string
}

func lockMarketCandleInvalidIssue(
	ctx context.Context,
	tx pgx.Tx,
	request data.QuarantineMarketCandleInvalidIssuesRequest,
	openTime time.Time,
	intervalDuration time.Duration,
) (marketCandleInvalidIssueRow, bool, error) {
	row := tx.QueryRow(ctx, `
		WITH scanned_candle AS (
			SELECT exchange, symbol, interval, open_time, close_time,
			       open::text, high::text, low::text, close::text, volume::text, is_closed,
			       CASE
			         WHEN open <= 0 THEN 'invalid_open_price'
			         WHEN high <= 0 THEN 'invalid_high_price'
			         WHEN low <= 0 THEN 'invalid_low_price'
			         WHEN close <= 0 THEN 'invalid_close_price'
			         WHEN volume < 0 THEN 'invalid_volume'
			         WHEN high < GREATEST(open, close, low) THEN 'invalid_high_bound'
			         WHEN low > LEAST(open, close, high) THEN 'invalid_low_bound'
			         WHEN date_bin($5::bigint * interval '1 second', open_time, TIMESTAMPTZ '1970-01-01 00:00:00+00') <> open_time THEN 'invalid_open_time'
			         WHEN close_time <> open_time + ($5::bigint * interval '1 second') THEN 'invalid_close_time'
			         ELSE NULL
			       END AS invalid_code,
			       CASE
			         WHEN open <= 0 THEN 'open price value must be positive'
			         WHEN high <= 0 THEN 'high price value must be positive'
			         WHEN low <= 0 THEN 'low price value must be positive'
			         WHEN close <= 0 THEN 'close price value must be positive'
			         WHEN volume < 0 THEN 'volume value is negative'
			         WHEN high < GREATEST(open, close, low) THEN 'high value is below OHLC bounds'
			         WHEN low > LEAST(open, close, high) THEN 'low value is above OHLC bounds'
			         WHEN date_bin($5::bigint * interval '1 second', open_time, TIMESTAMPTZ '1970-01-01 00:00:00+00') <> open_time THEN 'open time is not aligned to interval'
			         WHEN close_time <> open_time + ($5::bigint * interval '1 second') THEN 'close time does not match interval'
			         ELSE ''
			       END AS invalid_message
			  FROM market_candles
			 WHERE exchange = $1
			   AND symbol = $2
			   AND interval = $3
			   AND open_time = $4
			 FOR UPDATE
		)
		SELECT exchange, symbol, interval, open_time, close_time,
		       open, high, low, close, volume, is_closed, invalid_code, invalid_message
		  FROM scanned_candle
		 WHERE invalid_code IS NOT NULL`,
		request.Exchange,
		request.Symbol,
		request.Interval,
		openTime,
		int64(intervalDuration/time.Second),
	)

	var result marketCandleInvalidIssueRow
	if err := row.Scan(
		&result.candle.Exchange,
		&result.candle.Symbol,
		&result.candle.Interval,
		&result.candle.OpenTime,
		&result.candle.CloseTime,
		&result.candle.Open,
		&result.candle.High,
		&result.candle.Low,
		&result.candle.Close,
		&result.candle.Volume,
		&result.candle.IsClosed,
		&result.code,
		&result.message,
	); err != nil {
		if err == pgx.ErrNoRows {
			return marketCandleInvalidIssueRow{}, false, nil
		}
		return marketCandleInvalidIssueRow{}, false, fmt.Errorf("lock market candle invalid issue: %w", err)
	}
	result.candle.OpenTime = result.candle.OpenTime.UTC()
	result.candle.CloseTime = result.candle.CloseTime.UTC()
	return result, true, nil
}

func quarantineMarketCandle(
	ctx context.Context,
	tx pgx.Tx,
	row marketCandleInvalidIssueRow,
) (data.MarketCandleQuarantineRecord, error) {
	var quarantinedAt time.Time
	if err := tx.QueryRow(ctx, `
		INSERT INTO market_candle_quarantines (
			exchange, symbol, interval, open_time, close_time,
			open, high, low, close, volume, is_closed, reason, message
		)
		VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8::numeric, $9::numeric, $10::numeric, $11, $12, $13)
		ON CONFLICT (exchange, symbol, interval, open_time, reason)
		DO UPDATE SET
			close_time = EXCLUDED.close_time,
			open = EXCLUDED.open,
			high = EXCLUDED.high,
			low = EXCLUDED.low,
			close = EXCLUDED.close,
			volume = EXCLUDED.volume,
			is_closed = EXCLUDED.is_closed,
			message = EXCLUDED.message,
			quarantined_at = now()
		RETURNING quarantined_at`,
		row.candle.Exchange,
		row.candle.Symbol,
		row.candle.Interval,
		row.candle.OpenTime,
		row.candle.CloseTime,
		row.candle.Open,
		row.candle.High,
		row.candle.Low,
		row.candle.Close,
		row.candle.Volume,
		row.candle.IsClosed,
		row.code,
		row.message,
	).Scan(&quarantinedAt); err != nil {
		return data.MarketCandleQuarantineRecord{}, fmt.Errorf("archive quarantined market candle: %w", err)
	}

	commandTag, err := tx.Exec(ctx, `
		DELETE FROM market_candles
		 WHERE exchange = $1
		   AND symbol = $2
		   AND interval = $3
		   AND open_time = $4`,
		row.candle.Exchange,
		row.candle.Symbol,
		row.candle.Interval,
		row.candle.OpenTime,
	)
	if err != nil {
		return data.MarketCandleQuarantineRecord{}, fmt.Errorf("delete quarantined market candle: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return data.MarketCandleQuarantineRecord{}, data.ErrNotFound
	}

	return data.MarketCandleQuarantineRecord{
		Exchange:      row.candle.Exchange,
		Symbol:        row.candle.Symbol,
		Interval:      row.candle.Interval,
		OpenTime:      row.candle.OpenTime,
		CloseTime:     row.candle.CloseTime,
		Reason:        row.code,
		Message:       row.message,
		QuarantinedAt: quarantinedAt.UTC(),
	}, nil
}
