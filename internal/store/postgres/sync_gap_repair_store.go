package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

const maxDataSyncGapRepairTasks = 20

type dataSyncGapRepairWindow struct {
	from           time.Time
	to             time.Time
	missingCandles int
}

type dataSyncGapQueryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func (store *Store) ListDataSyncTaskGaps(ctx context.Context, id string) (data.DataSyncGapList, error) {
	task, err := getDataSyncTask(ctx, store.pool, id)
	if err != nil {
		return data.DataSyncGapList{}, err
	}

	windows, totalCount, limited, err := listDataSyncRepairWindows(ctx, store.pool, id)
	if err != nil {
		return data.DataSyncGapList{}, err
	}
	return data.DataSyncGapList{
		TaskID:        task.ID,
		Gaps:          dataSyncRepairWindowsToGaps(windows),
		Limited:       limited,
		TotalCount:    totalCount,
		ReturnedCount: len(windows),
		RepairLimit:   maxDataSyncGapRepairTasks,
	}, nil
}

func (store *Store) RepairDataSyncTaskGaps(
	ctx context.Context,
	id string,
) (data.DataSyncGapRepairResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("begin repair data sync gaps: %w", err)
	}
	defer tx.Rollback(ctx)

	source, err := lockDataSyncTask(ctx, tx, id)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	windows, totalCount, limited, err := listDataSyncRepairWindows(ctx, tx, id)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	result := data.DataSyncGapRepairResult{
		SourceTaskID: source.ID,
		CreatedTasks: []data.DataSyncTask{},
		Limited:      limited,
		TotalCount:   totalCount,
		RepairLimit:  maxDataSyncGapRepairTasks,
	}
	for _, window := range windows {
		exists, err := dataSyncRepairTaskExists(ctx, tx, source, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if exists {
			result.SkippedExisting++
			continue
		}

		task, err := insertDataSyncRepairTask(ctx, tx, source, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result.CreatedTasks = append(result.CreatedTasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("commit repair data sync gaps: %w", err)
	}
	return result, nil
}

func getDataSyncTask(ctx context.Context, queryer dataSyncGapQueryer, id string) (data.DataSyncTask, error) {
	row := queryer.QueryRow(ctx, `
		SELECT `+dataSyncTaskReturningColumns()+`
		  FROM data_sync_tasks
		 WHERE id = $1`,
		id,
	)
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.DataSyncTask{}, data.ErrNotFound
		}
		return data.DataSyncTask{}, fmt.Errorf("get data sync task: %w", err)
	}
	return task, nil
}

func lockDataSyncTask(ctx context.Context, tx pgx.Tx, id string) (data.DataSyncTask, error) {
	row := tx.QueryRow(ctx, `
		SELECT `+dataSyncTaskReturningColumns()+`
		  FROM data_sync_tasks
		 WHERE id = $1
		 FOR UPDATE`,
		id,
	)
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.DataSyncTask{}, data.ErrNotFound
		}
		return data.DataSyncTask{}, fmt.Errorf("lock data sync task: %w", err)
	}
	return task, nil
}

func listDataSyncRepairWindows(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	id string,
) ([]dataSyncGapRepairWindow, int, bool, error) {
	rows, err := queryer.Query(ctx, fmt.Sprintf(`
		WITH gaps AS (
			SELECT expected_open_time, open_time, missing_candles
			  FROM (
			SELECT open_time,
			       previous_open_time + interval_duration AS expected_open_time,
			       CASE
			         WHEN previous_open_time IS NULL OR interval_duration IS NULL THEN 0
			         ELSE GREATEST(
			           (EXTRACT(EPOCH FROM (open_time - previous_open_time))
			            / NULLIF(EXTRACT(EPOCH FROM interval_duration), 0))::int - 1,
			           0
			         )
			       END AS missing_candles,
			       previous_open_time IS NOT NULL
			       AND interval_duration IS NOT NULL
			       AND open_time - previous_open_time > interval_duration AS is_gap
			  FROM (
				SELECT c.open_time,
				       LAG(c.open_time) OVER (ORDER BY c.open_time) AS previous_open_time,
				       %s AS interval_duration
				  FROM data_sync_tasks AS t
				  JOIN market_candles AS c
				    ON c.exchange = t.exchange
				   AND c.symbol = t.symbol
				   AND c.interval = t.interval
				 WHERE t.id = $1
				   AND (t.start_time IS NULL OR c.open_time >= t.start_time)
				   AND c.open_time <= COALESCE(t.end_time, t.last_synced_open_time, now())
			  ) ordered_candles
			  ) gap_candidates
			 WHERE is_gap
		)
		SELECT expected_open_time, open_time, missing_candles, COUNT(*) OVER () AS total_count
		  FROM gaps
		 ORDER BY open_time
		 LIMIT $2`,
		dataSyncTaskIntervalDurationSQL("t"),
	), id, maxDataSyncGapRepairTasks)
	if err != nil {
		return nil, 0, false, fmt.Errorf("list data sync repair windows: %w", err)
	}
	defer rows.Close()

	windows := make([]dataSyncGapRepairWindow, 0)
	totalCount := 0
	for rows.Next() {
		var window dataSyncGapRepairWindow
		if err := rows.Scan(&window.from, &window.to, &window.missingCandles, &totalCount); err != nil {
			return nil, 0, false, fmt.Errorf("scan data sync repair window: %w", err)
		}
		windows = append(windows, window)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, false, fmt.Errorf("iterate data sync repair windows: %w", err)
	}

	limited := totalCount > maxDataSyncGapRepairTasks
	return windows, totalCount, limited, nil
}

func dataSyncRepairWindowsToGaps(windows []dataSyncGapRepairWindow) []data.CandleGap {
	gaps := make([]data.CandleGap, 0, len(windows))
	for _, window := range windows {
		gaps = append(gaps, data.CandleGap{
			From:           window.from,
			To:             window.to,
			MissingCandles: window.missingCandles,
		})
	}
	return gaps
}

func dataSyncRepairTaskExists(
	ctx context.Context,
	tx pgx.Tx,
	source data.DataSyncTask,
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
		)`,
		source.Exchange,
		source.Symbol,
		source.Interval,
		window.from,
		window.to,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check existing data sync repair task: %w", err)
	}
	return exists, nil
}

func insertDataSyncRepairTask(
	ctx context.Context,
	tx pgx.Tx,
	source data.DataSyncTask,
	window dataSyncGapRepairWindow,
) (data.DataSyncTask, error) {
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
		source.Exchange,
		source.Symbol,
		source.Interval,
		window.from,
		window.to,
		data.TaskStatusPending,
	)
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		return data.DataSyncTask{}, fmt.Errorf("insert data sync repair task: %w", err)
	}
	return task, nil
}
