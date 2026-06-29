package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
		 LIMIT $4`,
		query.Exchange,
		query.Symbol,
		query.Interval,
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
