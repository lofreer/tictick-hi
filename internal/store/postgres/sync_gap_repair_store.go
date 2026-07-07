package postgres

import (
	"context"
	"fmt"
	"strings"
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
	request data.RepairDataSyncTaskGapsRequest,
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
	if err := data.ValidateDataSyncTaskWindow(source.Interval, nil, nil); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if err := ensureDataSyncRepairSourceMarketActive(ctx, tx, source); err != nil {
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

		task, err := insertDataSyncRepairTask(ctx, tx, source, window, request.RequestID)
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

func (store *Store) RepairDataSyncTaskGap(
	ctx context.Context,
	id string,
	request data.RepairDataSyncTaskGapRequest,
) (data.DataSyncGapRepairResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("begin repair data sync gap: %w", err)
	}
	defer tx.Rollback(ctx)

	source, err := lockDataSyncTask(ctx, tx, id)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if err := data.ValidateDataSyncTaskWindow(source.Interval, nil, nil); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if err := ensureDataSyncRepairSourceMarketActive(ctx, tx, source); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	request.From = request.From.UTC()
	request.To = request.To.UTC()
	if err := data.ValidateDataSyncTaskWindow(source.Interval, &request.From, &request.To); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	window := dataSyncGapRepairWindow{from: request.From, to: request.To}
	ok, err := dataSyncRepairWindowExists(ctx, tx, source.ID, window)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if !ok {
		return data.DataSyncGapRepairResult{}, data.ErrNotFound
	}

	result := data.DataSyncGapRepairResult{
		SourceTaskID: source.ID,
		CreatedTasks: []data.DataSyncTask{},
		TotalCount:   1,
		RepairLimit:  1,
	}
	exists, err := dataSyncRepairTaskExists(ctx, tx, source, window)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if exists {
		result.SkippedExisting = 1
	} else {
		task, err := insertDataSyncRepairTask(ctx, tx, source, window, request.RequestID)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result.CreatedTasks = append(result.CreatedTasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("commit repair data sync gap: %w", err)
	}
	return result, nil
}

func getDataSyncTask(ctx context.Context, queryer dataSyncGapQueryer, id string) (data.DataSyncTask, error) {
	row := queryer.QueryRow(ctx, `
		SELECT `+dataSyncTaskReturningColumns()+`
		  FROM data_sync_tasks
		 WHERE id = $1
		   AND deleted_at IS NULL`,
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
		   AND deleted_at IS NULL
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

func ensureDataSyncRepairSourceMarketActive(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	source data.DataSyncTask,
) error {
	return ensureRepairMarketActive(ctx, queryer, source.Exchange, source.Symbol)
}

func ensureRepairMarketActive(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	exchange string,
	symbol string,
) error {
	var active bool
	if err := queryer.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			  FROM market_instruments
			 WHERE exchange = $1
			   AND symbol = $2
			   AND status = 'active'
		)`,
		exchange,
		strings.ToUpper(strings.TrimSpace(symbol)),
	).Scan(&active); err != nil {
		return fmt.Errorf("check repair market instrument: %w", err)
	}
	if !active {
		return data.MarketInstrumentNotActiveError()
	}
	return nil
}

func listDataSyncRepairWindows(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	id string,
) ([]dataSyncGapRepairWindow, int, bool, error) {
	rows, err := queryer.Query(ctx, fmt.Sprintf(`
		WITH %s
		SELECT gap_from, gap_to, missing_candles, COUNT(*) OVER () AS total_count
		  FROM gaps
		 ORDER BY gap_from, gap_to
		 LIMIT $2`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
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

func dataSyncRepairWindowExists(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	id string,
	window dataSyncGapRepairWindow,
) (bool, error) {
	var exists bool
	if err := queryer.QueryRow(ctx, fmt.Sprintf(`
		WITH %s
		SELECT EXISTS (
			SELECT 1
			  FROM gaps
			 WHERE gap_from = $2
			   AND gap_to = $3
		)`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
	), id, window.from, window.to).Scan(&exists); err != nil {
		return false, fmt.Errorf("check data sync repair window: %w", err)
	}
	return exists, nil
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
			   AND deleted_at IS NULL
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
	requestID string,
) (data.DataSyncTask, error) {
	from := window.from.UTC()
	to := window.to.UTC()
	if err := data.ValidateDataSyncTaskWindow(source.Interval, &from, &to); err != nil {
		return data.DataSyncTask{}, err
	}
	id, err := core.NewPrefixedID("dst")
	if err != nil {
		return data.DataSyncTask{}, err
	}
	row := tx.QueryRow(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, start_time, end_time,
			repair_source_task_id, sync_enabled, realtime_enabled, status, request_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, false, $8, NULLIF($9, ''))
		RETURNING `+dataSyncTaskReturningColumns(),
		id,
		source.Exchange,
		source.Symbol,
		source.Interval,
		from,
		to,
		source.ID,
		data.TaskStatusPending,
		requestID,
	)
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		return data.DataSyncTask{}, fmt.Errorf("insert data sync repair task: %w", err)
	}
	return task, nil
}
