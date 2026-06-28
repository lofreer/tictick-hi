package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/secretbox"
)

type Store struct {
	pool      *pgxpool.Pool
	secretBox *secretbox.Box
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	box, err := secretbox.FromEnv()
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Store{pool: pool, secretBox: box}, nil
}

func (store *Store) Close() {
	store.pool.Close()
}

func (store *Store) ListDataSyncTasks(ctx context.Context) ([]data.DataSyncTask, error) {
	rows, err := store.pool.Query(ctx, fmt.Sprintf(`
		SELECT %s
		  FROM data_sync_tasks AS t
		  LEFT JOIN LATERAL (
			SELECT COUNT(*) AS candle_count,
			       COALESCE(
			         BOOL_OR(
			           previous_open_time IS NOT NULL
			           AND interval_duration IS NOT NULL
			           AND open_time - previous_open_time > interval_duration
			         ),
			         false
			       ) AS has_gap
			  FROM (
				SELECT c.open_time,
				       LAG(c.open_time) OVER (ORDER BY c.open_time) AS previous_open_time,
				       %s AS interval_duration
				  FROM market_candles AS c
				 WHERE c.exchange = t.exchange
				   AND c.symbol = t.symbol
				   AND c.interval = t.interval
				   AND (t.start_time IS NULL OR c.open_time >= t.start_time)
				   AND c.open_time <= COALESCE(t.end_time, t.last_synced_open_time, now())
			  ) ordered_candles
		  ) candle_state ON true
		 ORDER BY t.created_at DESC`,
		dataSyncTaskScanColumns("t", dataSyncTaskListHealthSQL("t")),
		dataSyncTaskIntervalDurationSQL("t"),
	))
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
		RETURNING `+dataSyncTaskReturningColumns(),
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

func (store *Store) RetryDataSyncTask(ctx context.Context, id string) (data.DataSyncTask, error) {
	row := store.pool.QueryRow(ctx, fmt.Sprintf(`
		UPDATE data_sync_tasks
		   SET sync_enabled = true,
		       status = $2,
		       %s,
		       next_attempt_at = NULL,
		       finished_at = NULL,
		       last_error = NULL,
		       updated_at = now()
		 WHERE id = $1
		   AND status = $3
		RETURNING `+dataSyncTaskReturningColumns(),
		clearLeaseAssignments(dataSyncTaskLease)),
		id,
		data.TaskStatusPending,
		data.TaskStatusFailed,
	)
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			if exists, existsErr := store.dataSyncTaskExists(ctx, id); existsErr != nil {
				return data.DataSyncTask{}, existsErr
			} else if exists {
				return data.DataSyncTask{}, data.DataSyncRetryRequiresFailedError()
			}
			return data.DataSyncTask{}, data.ErrNotFound
		}
		return data.DataSyncTask{}, fmt.Errorf("retry data sync task: %w", err)
	}
	return task, nil
}

func (store *Store) dataSyncTaskExists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := store.pool.QueryRow(
		ctx,
		`SELECT EXISTS (SELECT 1 FROM data_sync_tasks WHERE id = $1)`,
		id,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check data sync task exists: %w", err)
	}
	return exists, nil
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
	result, err := store.GetCandles(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Candles, nil
}

func (store *Store) GetCandles(ctx context.Context, query data.CandleQuery) (data.CandleResult, error) {
	return data.NewCandleProvider(store).GetCandles(ctx, query)
}

func (store *Store) ListNativeCandles(ctx context.Context, query data.CandleQuery) ([]data.Candle, error) {
	limit := data.NormalizeCandleLimit(query.Limit)

	if query.From == nil && query.To == nil {
		return store.listLatestNativeCandles(ctx, query, limit)
	}
	return store.listNativeCandlesInRange(ctx, query, limit)
}

func (store *Store) listNativeCandlesInRange(
	ctx context.Context,
	query data.CandleQuery,
	limit int,
) ([]data.Candle, error) {
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

func (store *Store) listLatestNativeCandles(
	ctx context.Context,
	query data.CandleQuery,
	limit int,
) ([]data.Candle, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT exchange, symbol, interval, open_time, close_time,
		       open, high, low, close, volume, is_closed
		  FROM (
			SELECT exchange, symbol, interval, open_time, close_time,
			       open::text AS open, high::text AS high, low::text AS low,
			       close::text AS close, volume::text AS volume, is_closed
			  FROM market_candles
			 WHERE exchange = $1
			   AND symbol = $2
			   AND interval = $3
			 ORDER BY open_time DESC
			 LIMIT $4
		  ) latest
		 ORDER BY open_time ASC`,
		query.Exchange, query.Symbol, query.Interval, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list latest candles: %w", err)
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
		   SET %s = $2,
		       status = $3,
		       %s,
		       next_attempt_at = CASE WHEN $2::boolean THEN NULL ELSE next_attempt_at END,
		       finished_at = CASE WHEN $2::boolean THEN finished_at ELSE now() END,
		       updated_at = now()
		 WHERE id = $1
		   AND (
		     status = $3
		     OR ($3 IN ($4, $5, $6) AND status IN ($4, $5, $6))
		   )
		RETURNING `+dataSyncTaskReturningColumns(),
		column,
		clearLeaseCaseAssignments(dataSyncTaskLease, "NOT $2::boolean")),
		id,
		enabled,
		status,
		data.TaskStatusPending,
		data.TaskStatusRunning,
		data.TaskStatusPaused,
	)

	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			if exists, existsErr := store.dataSyncTaskExists(ctx, id); existsErr != nil {
				return data.DataSyncTask{}, existsErr
			} else if exists {
				return data.DataSyncTask{}, data.DataSyncCommandInvalidStateError()
			}
			return data.DataSyncTask{}, data.ErrNotFound
		}
		return data.DataSyncTask{}, fmt.Errorf("update data sync task: %w", err)
	}
	return task, nil
}

func dataSyncTaskReturningColumns() string {
	return dataSyncTaskScanColumns("", dataSyncTaskStateHealthSQL(""))
}

func dataSyncTaskScanColumns(alias string, healthSQL string) string {
	return fmt.Sprintf(`%s, %s AS data_health, %s, %s, %s, %s, %s, %s`,
		dataSyncTaskColumnList(alias, "id", "exchange", "symbol", "interval", "start_time", "end_time",
			"sync_enabled", "realtime_enabled", "status"),
		healthSQL,
		dataSyncTaskColumn(alias, "last_synced_open_time"),
		fmt.Sprintf("COALESCE(%s, '')", dataSyncTaskColumn(alias, "last_error")),
		dataSyncTaskColumn(alias, "attempt_count"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "created_at"),
		dataSyncTaskColumn(alias, "updated_at"),
	)
}

func dataSyncTaskColumnList(alias string, columns ...string) string {
	result := ""
	for index, column := range columns {
		if index > 0 {
			result += ", "
		}
		result += dataSyncTaskColumn(alias, column)
	}
	return result
}

func dataSyncTaskColumn(alias string, column string) string {
	if alias == "" {
		return column
	}
	return alias + "." + column
}

func dataSyncTaskListHealthSQL(alias string) string {
	return fmt.Sprintf(`CASE
		WHEN %s IN ('failed', 'cancelled') THEN 'failed'
		WHEN %s IS NOT NULL AND %s > now() THEN 'retrying'
		WHEN candle_state.has_gap THEN 'gap'
		WHEN %s = 'paused' THEN 'paused'
		WHEN %s IS NULL OR COALESCE(candle_state.candle_count, 0) = 0 THEN
			CASE WHEN (%s IN ('pending', 'running') AND (%s OR %s)) THEN 'syncing' ELSE 'insufficient' END
		WHEN %s IN ('pending', 'running') AND (%s OR %s) THEN 'syncing'
		ELSE 'ok'
	END`,
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "last_synced_open_time"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
	)
}

func dataSyncTaskStateHealthSQL(alias string) string {
	return fmt.Sprintf(`CASE
		WHEN %s IN ('failed', 'cancelled') THEN 'failed'
		WHEN %s IS NOT NULL AND %s > now() THEN 'retrying'
		WHEN %s = 'paused' THEN 'paused'
		WHEN %s IS NULL THEN
			CASE WHEN (%s IN ('pending', 'running') AND (%s OR %s)) THEN 'syncing' ELSE 'insufficient' END
		WHEN %s IN ('pending', 'running') AND (%s OR %s) THEN 'syncing'
		ELSE 'ok'
	END`,
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "last_synced_open_time"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
	)
}

func dataSyncTaskIntervalDurationSQL(alias string) string {
	intervalColumn := dataSyncTaskColumn(alias, "interval")
	return fmt.Sprintf(`CASE %s
		WHEN '1m' THEN interval '1 minute'
		WHEN '5m' THEN interval '5 minutes'
		WHEN '15m' THEN interval '15 minutes'
		WHEN '1h' THEN interval '1 hour'
		WHEN '4h' THEN interval '4 hours'
		WHEN '1d' THEN interval '1 day'
		ELSE NULL
	END`, intervalColumn)
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
		&task.DataHealth,
		&task.LatestSyncedOpenTime,
		&task.LastError,
		&task.AttemptCount,
		&task.NextAttemptAt,
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
