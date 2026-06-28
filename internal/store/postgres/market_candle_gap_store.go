package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ScanMarketCandleGaps(
	ctx context.Context,
	query data.MarketCandleGapScanQuery,
) (data.MarketCandleGapScan, error) {
	intervalDuration, err := data.IntervalDuration(query.Interval)
	if err != nil {
		return data.MarketCandleGapScan{}, err
	}
	limit := data.NormalizeMarketCandleGapScanLimit(query.Limit)

	result, err := store.marketCandleGapScanWindow(ctx, query)
	if err != nil {
		return data.MarketCandleGapScan{}, err
	}
	result.Exchange = query.Exchange
	result.Symbol = query.Symbol
	result.Interval = query.Interval
	result.Gaps = []data.CandleGap{}

	if result.Window.Count < 2 {
		return result, nil
	}

	gaps, totalCount, err := store.marketCandleGapScanRows(ctx, query, intervalDuration, limit)
	if err != nil {
		return data.MarketCandleGapScan{}, err
	}
	result.Gaps = gaps
	result.TotalCount = totalCount
	result.ReturnedCount = len(gaps)
	result.Limited = totalCount > len(gaps)
	return result, nil
}

func (store *Store) marketCandleGapScanWindow(
	ctx context.Context,
	query data.MarketCandleGapScanQuery,
) (data.MarketCandleGapScan, error) {
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
		return data.MarketCandleGapScan{}, fmt.Errorf("scan market candle gap window: %w", err)
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
	return data.MarketCandleGapScan{Window: window}, nil
}

func (store *Store) marketCandleGapScanRows(
	ctx context.Context,
	query data.MarketCandleGapScanQuery,
	intervalDuration time.Duration,
	limit int,
) ([]data.CandleGap, int, error) {
	intervalSeconds := int64(intervalDuration / time.Second)
	rows, err := store.pool.Query(ctx, `
		WITH ordered AS (
			SELECT open_time,
			       LAG(open_time) OVER (ORDER BY open_time) AS previous_open_time
			  FROM market_candles
			 WHERE exchange = $1
			   AND symbol = $2
			   AND interval = $3
		),
		gaps AS (
			SELECT previous_open_time + ($4::bigint * interval '1 second') AS gap_from,
			       open_time AS gap_to,
			       GREATEST(
			         (EXTRACT(EPOCH FROM (open_time - previous_open_time)) / NULLIF($4::numeric, 0))::int - 1,
			         0
			       ) AS missing_candles
			  FROM ordered
			 WHERE previous_open_time IS NOT NULL
			   AND open_time - previous_open_time > ($4::bigint * interval '1 second')
		)
		SELECT gap_from, gap_to, missing_candles, COUNT(*) OVER ()::int AS total_count
		  FROM gaps
		 ORDER BY gap_to ASC
		 LIMIT $5`,
		query.Exchange,
		query.Symbol,
		query.Interval,
		intervalSeconds,
		limit,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("scan market candle gaps: %w", err)
	}
	defer rows.Close()

	gaps := make([]data.CandleGap, 0)
	totalCount := 0
	for rows.Next() {
		var gap data.CandleGap
		if err := rows.Scan(&gap.From, &gap.To, &gap.MissingCandles, &totalCount); err != nil {
			return nil, 0, fmt.Errorf("scan market candle gap row: %w", err)
		}
		gap.From = gap.From.UTC()
		gap.To = gap.To.UTC()
		gaps = append(gaps, gap)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate market candle gaps: %w", err)
	}
	return gaps, totalCount, nil
}
