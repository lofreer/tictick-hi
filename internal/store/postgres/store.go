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
			%s
		  ) candle_state ON true
		 ORDER BY t.created_at DESC`,
		dataSyncTaskScanColumns("t", dataSyncTaskListHealthSQL("t"), dataSyncTaskListGapSummarySQL()),
		dataSyncTaskCandleStateLateralSQL(),
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
		return store.ListLatestNativeCandles(ctx, query)
	}
	return store.listNativeCandlesInRange(ctx, query, limit)
}

func (store *Store) ListLatestNativeCandles(ctx context.Context, query data.CandleQuery) ([]data.Candle, error) {
	limit := data.NormalizeCandleLimit(query.Limit)
	return store.listLatestNativeCandles(ctx, query, limit)
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
			   AND ($4::timestamptz IS NULL OR open_time >= $4)
			   AND ($5::timestamptz IS NULL OR open_time <= $5)
			 ORDER BY open_time DESC
			 LIMIT $6
		  ) latest
		 ORDER BY open_time ASC`,
		query.Exchange, query.Symbol, query.Interval, query.From, query.To, limit,
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
