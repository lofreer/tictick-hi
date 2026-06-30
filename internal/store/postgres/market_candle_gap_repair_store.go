package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) RepairMarketCandleGap(
	ctx context.Context,
	request data.RepairMarketCandleGapRequest,
) (data.DataSyncGapRepairResult, error) {
	request.From = request.From.UTC()
	request.To = request.To.UTC()
	if err := data.ValidateDataSyncTaskWindow(request.Interval, &request.From, &request.To); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	intervalDuration, err := data.IntervalDuration(request.Interval)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("begin repair market candle gap: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureRepairMarketActive(ctx, tx, request.Exchange, request.Symbol); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	window, ok, err := marketCandleRepairWindow(ctx, tx, request, intervalDuration)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if !ok {
		return data.DataSyncGapRepairResult{}, data.ErrNotFound
	}

	result := data.DataSyncGapRepairResult{
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   1,
		RepairLimit:  1,
	}
	exists, err := marketCandleRepairTaskExists(ctx, tx, request, window)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if exists {
		result.SkippedExisting = 1
	} else {
		task, err := insertMarketCandleRepairTask(ctx, tx, request, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result.CreatedTasks = append(result.CreatedTasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("commit repair market candle gap: %w", err)
	}
	return result, nil
}

func (store *Store) RepairMarketCandleGaps(
	ctx context.Context,
	request data.RepairMarketCandleGapsRequest,
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
		return data.DataSyncGapRepairResult{}, fmt.Errorf("begin repair market candle gaps: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := ensureRepairMarketActive(ctx, tx, request.Exchange, request.Symbol); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	result := data.DataSyncGapRepairResult{
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   len(request.Gaps),
		RepairLimit:  data.MaxMarketCandleGapScanLimit,
	}
	for _, gap := range request.Gaps {
		from := gap.From.UTC()
		to := gap.To.UTC()
		if err := data.ValidateDataSyncTaskWindow(request.Interval, &from, &to); err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		gapRequest := data.RepairMarketCandleGapRequest{
			Exchange: request.Exchange,
			Symbol:   request.Symbol,
			Interval: request.Interval,
			From:     from,
			To:       to,
		}
		window, ok, err := marketCandleRepairWindow(ctx, tx, gapRequest, intervalDuration)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if !ok {
			return data.DataSyncGapRepairResult{}, data.ErrNotFound
		}
		exists, err := marketCandleRepairTaskExists(ctx, tx, gapRequest, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if exists {
			result.SkippedExisting += 1
			continue
		}
		task, err := insertMarketCandleRepairTask(ctx, tx, gapRequest, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result.CreatedTasks = append(result.CreatedTasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("commit repair market candle gaps: %w", err)
	}
	return result, nil
}

func marketCandleRepairWindow(
	ctx context.Context,
	tx pgx.Tx,
	request data.RepairMarketCandleGapRequest,
	intervalDuration time.Duration,
) (dataSyncGapRepairWindow, bool, error) {
	intervalSeconds := int64(intervalDuration / time.Second)
	row := tx.QueryRow(ctx, `
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
		SELECT gap_from, gap_to, missing_candles
		  FROM gaps
		 WHERE gap_from = $5
		   AND gap_to = $6`,
		request.Exchange,
		request.Symbol,
		request.Interval,
		intervalSeconds,
		request.From,
		request.To,
	)

	var window dataSyncGapRepairWindow
	if err := row.Scan(&window.from, &window.to, &window.missingCandles); err != nil {
		if err == pgx.ErrNoRows {
			return dataSyncGapRepairWindow{}, false, nil
		}
		return dataSyncGapRepairWindow{}, false, fmt.Errorf("find market candle repair window: %w", err)
	}
	window.from = window.from.UTC()
	window.to = window.to.UTC()
	return window, true, nil
}

func marketCandleRepairTaskExists(
	ctx context.Context,
	tx pgx.Tx,
	request data.RepairMarketCandleGapRequest,
	window dataSyncGapRepairWindow,
) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM data_sync_tasks
			 WHERE exchange = $1
			   AND symbol = $2
			   AND interval = $3
			   AND start_time = $4
			   AND end_time = $5
			   AND deleted_at IS NULL
		)`,
		request.Exchange,
		request.Symbol,
		request.Interval,
		window.from,
		window.to,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check existing market candle repair task: %w", err)
	}
	return exists, nil
}

func insertMarketCandleRepairTask(
	ctx context.Context,
	tx pgx.Tx,
	request data.RepairMarketCandleGapRequest,
	window dataSyncGapRepairWindow,
) (data.DataSyncTask, error) {
	from := window.from.UTC()
	to := window.to.UTC()
	if err := data.ValidateDataSyncTaskWindow(request.Interval, &from, &to); err != nil {
		return data.DataSyncTask{}, err
	}
	id, err := core.NewPrefixedID("dst")
	if err != nil {
		return data.DataSyncTask{}, err
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, start_time, end_time,
			sync_enabled, realtime_enabled, status
		)
		VALUES ($1, $2, $3, $4, $5, $6, true, false, $7)
		RETURNING `+dataSyncTaskReturningColumns(),
		id,
		request.Exchange,
		request.Symbol,
		request.Interval,
		from,
		to,
		data.TaskStatusPending,
	)
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		return data.DataSyncTask{}, fmt.Errorf("insert market candle repair task: %w", err)
	}
	return task, nil
}
