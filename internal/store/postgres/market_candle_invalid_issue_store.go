package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ScanMarketCandleInvalidIssues(
	ctx context.Context,
	query data.MarketCandleInvalidIssueScanQuery,
) (data.MarketCandleInvalidIssueScan, error) {
	if _, err := data.IntervalDuration(query.Interval); err != nil {
		return data.MarketCandleInvalidIssueScan{}, err
	}
	limit := data.NormalizeMarketCandleInvalidIssueScanLimit(query.Limit)

	result, err := store.marketCandleInvalidIssueScanWindow(ctx, query)
	if err != nil {
		return data.MarketCandleInvalidIssueScan{}, err
	}
	result.Exchange = query.Exchange
	result.Symbol = query.Symbol
	result.Interval = query.Interval
	result.Issues = []data.CandleIssue{}

	if result.Window.Count == 0 {
		return result, nil
	}

	issues, totalCount, err := store.marketCandleInvalidIssueScanRows(ctx, query, limit)
	if err != nil {
		return data.MarketCandleInvalidIssueScan{}, err
	}
	result.Issues = issues
	result.TotalCount = totalCount
	result.ReturnedCount = len(issues)
	result.Limited = totalCount > len(issues)
	return result, nil
}

func (store *Store) marketCandleInvalidIssueScanWindow(
	ctx context.Context,
	query data.MarketCandleInvalidIssueScanQuery,
) (data.MarketCandleInvalidIssueScan, error) {
	var (
		from  sql.NullTime
		to    sql.NullTime
		count int
	)
	if err := store.pool.QueryRow(ctx, `
		SELECT MIN(open_time), MAX(open_time), COUNT(*)::int
		  FROM market_candles
		 WHERE exchange = $1
		   AND symbol = $2
		   AND interval = $3`,
		query.Exchange,
		query.Symbol,
		query.Interval,
	).Scan(&from, &to, &count); err != nil {
		return data.MarketCandleInvalidIssueScan{}, fmt.Errorf("scan market candle invalid issue window: %w", err)
	}

	window := data.CandleWindow{Count: count}
	if from.Valid {
		value := from.Time.UTC()
		window.From = &value
	}
	if to.Valid {
		value := to.Time.UTC()
		window.To = &value
	}
	return data.MarketCandleInvalidIssueScan{Window: window}, nil
}

func (store *Store) marketCandleInvalidIssueScanRows(
	ctx context.Context,
	query data.MarketCandleInvalidIssueScanQuery,
	limit int,
) ([]data.CandleIssue, int, error) {
	duration, err := data.IntervalDuration(query.Interval)
	if err != nil {
		return nil, 0, err
	}
	durationSeconds := int64(duration / time.Second)
	rows, err := store.pool.Query(ctx, `
		WITH scanned_candles AS (
			SELECT open_time,
			       CASE
			         WHEN open <= 0 THEN 'invalid_open_price'
			         WHEN high <= 0 THEN 'invalid_high_price'
			         WHEN low <= 0 THEN 'invalid_low_price'
			         WHEN close <= 0 THEN 'invalid_close_price'
			         WHEN volume < 0 THEN 'invalid_volume'
			         WHEN high < GREATEST(open, close, low) THEN 'invalid_high_bound'
			         WHEN low > LEAST(open, close, high) THEN 'invalid_low_bound'
			         WHEN date_bin($4::bigint * interval '1 second', open_time, TIMESTAMPTZ '1970-01-01 00:00:00+00') <> open_time THEN 'invalid_open_time'
			         WHEN close_time <> open_time + ($4::bigint * interval '1 second') THEN 'invalid_close_time'
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
			         WHEN date_bin($4::bigint * interval '1 second', open_time, TIMESTAMPTZ '1970-01-01 00:00:00+00') <> open_time THEN 'open time is not aligned to interval'
			         WHEN close_time <> open_time + ($4::bigint * interval '1 second') THEN 'close time does not match interval'
			         ELSE ''
			       END AS invalid_message
			  FROM market_candles
			 WHERE exchange = $1
			   AND symbol = $2
			   AND interval = $3
		)
		SELECT open_time, invalid_code, invalid_message, COUNT(*) OVER ()::int AS total_count
		  FROM scanned_candles
		 WHERE invalid_code IS NOT NULL
		 ORDER BY open_time ASC
		 LIMIT $5`,
		query.Exchange,
		query.Symbol,
		query.Interval,
		durationSeconds,
		limit,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("scan market candle invalid issues: %w", err)
	}
	defer rows.Close()

	issues := make([]data.CandleIssue, 0)
	totalCount := 0
	for rows.Next() {
		var openTime time.Time
		var issue data.CandleIssue
		if err := rows.Scan(&openTime, &issue.Code, &issue.Message, &totalCount); err != nil {
			return nil, 0, fmt.Errorf("scan market candle invalid issue row: %w", err)
		}
		value := openTime.UTC()
		issue.OpenTime = &value
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate market candle invalid issues: %w", err)
	}
	return issues, totalCount, nil
}

func (store *Store) RepairMarketCandleInvalidIssues(
	ctx context.Context,
	request data.RepairMarketCandleInvalidIssuesRequest,
) (data.DataSyncGapRepairResult, error) {
	if err := data.ValidateDataSyncTaskWindow(request.Interval, nil, nil); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	intervalDuration, err := data.IntervalDuration(request.Interval)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("begin repair market candle invalid issues: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureRepairMarketActive(ctx, tx, request.Exchange, request.Symbol); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	result := data.DataSyncGapRepairResult{
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   len(request.OpenTimes),
		RepairLimit:  data.MaxMarketCandleInvalidIssueScanLimit,
	}
	for _, rawOpenTime := range request.OpenTimes {
		openTime := rawOpenTime.UTC()
		window, ok, repairable, err := marketCandleInvalidIssueRepairWindow(ctx, tx, request, openTime, intervalDuration)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if !ok {
			return data.DataSyncGapRepairResult{}, data.ErrNotFound
		}
		if !repairable {
			continue
		}
		exists, err := marketCandleRepairTaskExists(ctx, tx, data.RepairMarketCandleGapRequest{
			Exchange:    request.Exchange,
			Symbol:      request.Symbol,
			Interval:    request.Interval,
			From:        window.from,
			To:          window.to,
			RequestID:   request.RequestID,
			TraceParent: request.TraceParent,
		}, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if exists {
			result.SkippedExisting++
			continue
		}
		task, err := insertMarketCandleRepairTask(ctx, tx, data.RepairMarketCandleGapRequest{
			Exchange:    request.Exchange,
			Symbol:      request.Symbol,
			Interval:    request.Interval,
			From:        window.from,
			To:          window.to,
			RequestID:   request.RequestID,
			TraceParent: request.TraceParent,
		}, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result.CreatedTasks = append(result.CreatedTasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("commit repair market candle invalid issues: %w", err)
	}
	return result, nil
}

func marketCandleInvalidIssueRepairWindow(
	ctx context.Context,
	tx pgx.Tx,
	request data.RepairMarketCandleInvalidIssuesRequest,
	openTime time.Time,
	intervalDuration time.Duration,
) (dataSyncGapRepairWindow, bool, bool, error) {
	row := tx.QueryRow(ctx, `
		WITH scanned_candle AS (
			SELECT open_time,
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
			       END AS invalid_code
			  FROM market_candles
			 WHERE exchange = $1
			   AND symbol = $2
			   AND interval = $3
			   AND open_time = $4
		)
		SELECT open_time, invalid_code
		  FROM scanned_candle
		 WHERE invalid_code IS NOT NULL`,
		request.Exchange,
		request.Symbol,
		request.Interval,
		openTime,
		int64(intervalDuration/time.Second),
	)

	var persistedOpenTime time.Time
	var invalidCode string
	if err := row.Scan(&persistedOpenTime, &invalidCode); err != nil {
		if err == pgx.ErrNoRows {
			return dataSyncGapRepairWindow{}, false, false, nil
		}
		return dataSyncGapRepairWindow{}, false, false, fmt.Errorf("find market candle invalid issue repair window: %w", err)
	}
	if !data.IsRepairableCandleIssueCode(invalidCode) {
		return dataSyncGapRepairWindow{}, true, false, nil
	}
	from := persistedOpenTime.UTC()
	return dataSyncGapRepairWindow{
		from:           from,
		to:             from.Add(intervalDuration),
		missingCandles: 1,
	}, true, true, nil
}
