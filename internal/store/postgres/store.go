package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (store *Store) Close() {
	store.pool.Close()
}

func (store *Store) ListDataSyncTasks(ctx context.Context) ([]data.DataSyncTask, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, exchange, symbol, interval, start_time, end_time,
		       sync_enabled, realtime_enabled, status, last_synced_open_time,
		       COALESCE(last_error, ''), attempt_count, created_at, updated_at
		  FROM data_sync_tasks
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list data sync tasks: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanDataSyncTask)
}

func (store *Store) CreateDataSyncTask(
	ctx context.Context,
	task data.CreateDataSyncTask,
) (data.DataSyncTask, error) {
	id, err := core.NewPrefixedID("dst")
	if err != nil {
		return data.DataSyncTask{}, err
	}

	row := store.pool.QueryRow(ctx, `
		INSERT INTO data_sync_tasks (id, exchange, symbol, interval, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, exchange, symbol, interval, start_time, end_time,
		          sync_enabled, realtime_enabled, status, last_synced_open_time,
		          COALESCE(last_error, ''), attempt_count, created_at, updated_at`,
		id, task.Exchange, task.Symbol, task.Interval, task.StartTime, task.EndTime,
	)

	created, err := scanDataSyncTaskRow(row)
	if err != nil {
		return data.DataSyncTask{}, fmt.Errorf("create data sync task: %w", err)
	}
	return created, nil
}

func (store *Store) DeleteDataSyncTask(ctx context.Context, id string) error {
	commandTag, err := store.pool.Exec(ctx, `DELETE FROM data_sync_tasks WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete data sync task: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return data.ErrNotFound
	}
	return nil
}

func (store *Store) SetSyncEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	status := data.TaskStatusPending
	if !enabled {
		status = data.TaskStatusPaused
	}
	return store.updateTaskFlag(ctx, id, "sync_enabled", enabled, status)
}

func (store *Store) SetRealtimeEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	status := data.TaskStatusRunning
	if !enabled {
		status = data.TaskStatusPaused
	}
	return store.updateTaskFlag(ctx, id, "realtime_enabled", enabled, status)
}

func (store *Store) ListCandles(ctx context.Context, query data.CandleQuery) ([]data.Candle, error) {
	limit := query.Limit
	if limit <= 0 || limit > 5000 {
		limit = 1000
	}

	rows, err := store.pool.Query(ctx, `
		SELECT exchange, symbol, interval, open_time, close_time,
		       open::text, high::text, low::text, close::text, volume::text, is_closed
		  FROM market_candles
		 WHERE exchange = $1
		   AND symbol = $2
		   AND interval = $3
		   AND ($4::timestamptz IS NULL OR open_time >= $4)
		   AND ($5::timestamptz IS NULL OR open_time <= $5)
		 ORDER BY open_time ASC
		 LIMIT $6`,
		query.Exchange, query.Symbol, query.Interval, query.From, query.To, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list candles: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanCandle)
}

func (store *Store) updateTaskFlag(
	ctx context.Context,
	id string,
	column string,
	enabled bool,
	status data.TaskStatus,
) (data.DataSyncTask, error) {
	row := store.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE data_sync_tasks
		   SET %s = $2, status = $3, updated_at = now()
		 WHERE id = $1
		RETURNING id, exchange, symbol, interval, start_time, end_time,
		          sync_enabled, realtime_enabled, status, last_synced_open_time,
		          COALESCE(last_error, ''), attempt_count, created_at, updated_at`, column),
		id, enabled, status,
	)

	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.DataSyncTask{}, data.ErrNotFound
		}
		return data.DataSyncTask{}, fmt.Errorf("update data sync task: %w", err)
	}
	return task, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanDataSyncTask(row pgx.CollectableRow) (data.DataSyncTask, error) {
	return scanDataSyncTaskRow(row)
}

func scanDataSyncTaskRow(row rowScanner) (data.DataSyncTask, error) {
	var task data.DataSyncTask
	err := row.Scan(
		&task.ID,
		&task.Exchange,
		&task.Symbol,
		&task.Interval,
		&task.StartTime,
		&task.EndTime,
		&task.SyncEnabled,
		&task.RealtimeEnabled,
		&task.Status,
		&task.LatestSyncedOpenTime,
		&task.LastError,
		&task.AttemptCount,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	return task, err
}

func scanCandle(row pgx.CollectableRow) (data.Candle, error) {
	var candle data.Candle
	err := row.Scan(
		&candle.Exchange,
		&candle.Symbol,
		&candle.Interval,
		&candle.OpenTime,
		&candle.CloseTime,
		&candle.Open,
		&candle.High,
		&candle.Low,
		&candle.Close,
		&candle.Volume,
		&candle.IsClosed,
	)
	return candle, err
}
