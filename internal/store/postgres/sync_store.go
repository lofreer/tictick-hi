package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/errtext"
)

func (store *Store) ClaimDataSyncTask(
	ctx context.Context,
	workerID string,
	leaseTTL time.Duration,
) (data.DataSyncTask, bool, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.DataSyncTask{}, false, fmt.Errorf("begin claim data sync task: %w", err)
	}
	defer tx.Rollback(ctx)

	row, ok, err := claimLeaseRow(
		ctx,
		tx,
		leaseClaimQuery{
			resource: dataSyncTaskLease,
			where: `(sync_enabled = true OR realtime_enabled = true)
				   AND deleted_at IS NULL
				   AND status IN ($1, $2)
				   AND (next_attempt_at IS NULL OR next_attempt_at <= now())
				   AND NOT EXISTS (
				     SELECT 1
				       FROM data_sync_exchange_backoffs
				      WHERE data_sync_exchange_backoffs.exchange = data_sync_tasks.exchange
				        AND data_sync_exchange_backoffs.next_attempt_at > now()
				   )
				   AND EXISTS (
				     SELECT 1
				       FROM market_instruments AS instrument
				      WHERE instrument.exchange = data_sync_tasks.exchange
				        AND instrument.symbol = data_sync_tasks.symbol
				        AND instrument.status = 'active'
				   )`,
			orderBy: `CASE
			           WHEN status = 'pending' THEN 0
			           WHEN sync_enabled = true THEN 1
			           WHEN realtime_enabled = true THEN 2
			           ELSE 3
			         END,
			         created_at ASC`,
			args: []any{
				data.TaskStatusPending,
				data.TaskStatusRunning,
			},
		},
		leaseClaimUpdate{
			resource:         dataSyncTaskLease,
			statusAssignment: "status = $2",
			workerArg:        "$3",
			ttlArg:           "$4",
			extraAssignments: []string{
				"started_at = COALESCE(started_at, now())",
				"next_attempt_at = NULL",
			},
			returningColumns: dataSyncTaskReturningColumns(),
		},
		data.TaskStatusRunning,
		workerID,
		intervalLiteral(leaseTTL),
	)
	if err != nil {
		return data.DataSyncTask{}, false, fmt.Errorf("claim data sync task: %w", err)
	}
	if !ok {
		return data.DataSyncTask{}, false, nil
	}
	task, err := scanDataSyncTaskRow(row)
	if err != nil {
		return data.DataSyncTask{}, false, fmt.Errorf("update claimed task: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncTask{}, false, fmt.Errorf("commit claim data sync task: %w", err)
	}
	return task, true, nil
}

func (store *Store) HeartbeatDataSyncTask(
	ctx context.Context,
	taskID string,
	workerID string,
	leaseTTL time.Duration,
) error {
	alive, err := heartbeatLease(
		ctx,
		store.pool,
		dataSyncTaskLease,
		taskID,
		workerID,
		intervalLiteral(leaseTTL),
		string(data.TaskStatusRunning),
	)
	if err != nil {
		return fmt.Errorf("heartbeat data sync task: %w", err)
	}
	if !alive {
		return data.DataSyncCommandInvalidStateError()
	}
	return nil
}

func (store *Store) SaveDataSyncResult(ctx context.Context, result data.DataSyncResult) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save sync result: %w", err)
	}
	defer tx.Rollback(ctx)

	target, err := readDataSyncTaskTarget(ctx, tx, result.TaskID, result.WorkerID)
	if err != nil {
		return err
	}
	if err := data.ValidateCandleSeriesForTarget(result.Candles, target.exchange, target.symbol, target.interval); err != nil {
		return fmt.Errorf("validate data sync result candles: %w", err)
	}

	for _, candle := range result.Candles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO market_candles (
				exchange, symbol, interval, open_time, close_time,
				open, high, low, close, volume, is_closed, updated_at
			)
			VALUES ($1, $2, $3, $4, $5, $6::numeric, $7::numeric, $8::numeric,
			        $9::numeric, $10::numeric, $11, now())
			ON CONFLICT (exchange, symbol, interval, open_time)
			DO UPDATE SET
				close_time = EXCLUDED.close_time,
				open = EXCLUDED.open,
				high = EXCLUDED.high,
				low = EXCLUDED.low,
				close = EXCLUDED.close,
				volume = EXCLUDED.volume,
				is_closed = EXCLUDED.is_closed,
				updated_at = now()`,
			candle.Exchange,
			candle.Symbol,
			candle.Interval,
			candle.OpenTime,
			candle.CloseTime,
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.IsClosed,
		); err != nil {
			return fmt.Errorf("upsert market candle: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: dataSyncTaskLease,
		assignments: []string{
			"last_synced_open_time = COALESCE($2, last_synced_open_time)",
			`status = CASE
			         WHEN realtime_enabled THEN $3
			         WHEN $4::boolean THEN $5
			         ELSE $6
			       END`,
			`sync_enabled = CASE
			         WHEN $4::boolean AND NOT realtime_enabled THEN false
			         ELSE sync_enabled
			       END`,
			`finished_at = CASE
			         WHEN $4::boolean AND NOT realtime_enabled THEN now()
			         ELSE finished_at
			       END`,
			"last_error = NULL",
			"next_attempt_at = NULL",
		},
		where: "id = $1 AND deleted_at IS NULL",
	}),
		result.TaskID,
		result.LastOpenTime,
		data.TaskStatusRunning,
		result.Completed,
		data.TaskStatusSucceeded,
		data.TaskStatusPending,
	); err != nil {
		return fmt.Errorf("update data sync task result: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		DELETE FROM data_sync_exchange_backoffs AS exchange_backoff
		 WHERE exchange_backoff.next_attempt_at <= now()
		   AND exchange_backoff.exchange = (
		       SELECT data_sync_tasks.exchange
		         FROM data_sync_tasks
		        WHERE data_sync_tasks.id = $1
		   )`,
		result.TaskID,
	); err != nil {
		return fmt.Errorf("clear data sync exchange backoff: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sync result: %w", err)
	}
	return nil
}

type dataSyncTaskTarget struct {
	exchange       string
	symbol         string
	interval       string
	status         data.TaskStatus
	hasActiveLease bool
}

func readDataSyncTaskTarget(ctx context.Context, tx pgx.Tx, taskID string, workerID string) (dataSyncTaskTarget, error) {
	var target dataSyncTaskTarget
	if err := tx.QueryRow(ctx, `
			SELECT exchange, symbol, interval, status,
			       locked_by = $2
			         AND locked_until IS NOT NULL
			         AND locked_until > now() AS has_active_lease
			  FROM data_sync_tasks
			 WHERE id = $1
			   AND deleted_at IS NULL
			 FOR UPDATE`,
		taskID,
		workerID,
	).Scan(
		&target.exchange,
		&target.symbol,
		&target.interval,
		&target.status,
		&target.hasActiveLease,
	); err != nil {
		if err == pgx.ErrNoRows {
			return dataSyncTaskTarget{}, data.ErrNotFound
		}
		return dataSyncTaskTarget{}, fmt.Errorf("read data sync task target: %w", err)
	}
	if target.status != data.TaskStatusRunning || !target.hasActiveLease {
		return dataSyncTaskTarget{}, data.DataSyncCommandInvalidStateError()
	}
	return target, nil
}

func (store *Store) MarkDataSyncFailed(ctx context.Context, taskID string, workerID string, taskErr error) error {
	commandTag, err := store.pool.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: dataSyncTaskLease,
		assignments: []string{
			"status = $2",
			"sync_enabled = false",
			"realtime_enabled = false",
			"last_error = $3",
			"next_attempt_at = NULL",
			"finished_at = now()",
		},
		where: dataSyncOwnedActiveLeaseWhere("$1", "$4"),
	}),
		taskID,
		data.TaskStatusFailed,
		normalizeTaskError(taskErr),
		workerID,
	)
	if err != nil {
		return fmt.Errorf("mark data sync failed: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return store.dataSyncTaskOwnershipError(ctx, taskID)
	}
	return nil
}

func (store *Store) RecordDataSyncRetry(
	ctx context.Context,
	taskID string,
	workerID string,
	taskErr error,
	nextAttemptAt *time.Time,
) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin record data sync retry: %w", err)
	}
	defer tx.Rollback(ctx)

	normalizedError := normalizeTaskError(taskErr)
	commandTag, err := tx.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: dataSyncTaskLease,
		assignments: []string{
			`status = CASE
			         WHEN realtime_enabled THEN $2
			         ELSE $3
			       END`,
			"last_error = $4",
			"next_attempt_at = $5",
		},
		where: dataSyncOwnedActiveLeaseWhere("$1", "$6"),
	}),
		taskID,
		data.TaskStatusRunning,
		data.TaskStatusPending,
		normalizedError,
		nextAttemptAt,
		workerID,
	)
	if err != nil {
		return fmt.Errorf("record data sync retry: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return store.dataSyncTaskOwnershipError(ctx, taskID)
	}

	if nextAttemptAt != nil {
		var exchangeName string
		if err := tx.QueryRow(ctx, `
			SELECT exchange
			  FROM data_sync_tasks
			 WHERE id = $1
			   AND deleted_at IS NULL`,
			taskID,
		).Scan(&exchangeName); err != nil {
			if err == pgx.ErrNoRows {
				return data.ErrNotFound
			}
			return fmt.Errorf("read data sync retry exchange: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO data_sync_exchange_backoffs (exchange, next_attempt_at, last_error, updated_at)
			VALUES ($1, $2, $3, now())
			ON CONFLICT (exchange)
			DO UPDATE SET
				next_attempt_at = GREATEST(data_sync_exchange_backoffs.next_attempt_at, EXCLUDED.next_attempt_at),
				last_error = EXCLUDED.last_error,
				updated_at = now()`,
			exchangeName,
			nextAttemptAt,
			normalizedError,
		); err != nil {
			return fmt.Errorf("record data sync exchange backoff: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit record data sync retry: %w", err)
	}
	return nil
}

func (store *Store) ReleaseDataSyncTask(ctx context.Context, taskID string, workerID string) error {
	commandTag, err := store.pool.Exec(ctx, fmt.Sprintf(`
			UPDATE data_sync_tasks
			   SET %s,
			       updated_at = now()
			 WHERE id = $1
			   AND deleted_at IS NULL
			   AND (locked_by = $2 OR locked_by IS NULL)`,
		clearLeaseAssignments(dataSyncTaskLease),
	), taskID, workerID)
	if err != nil {
		return fmt.Errorf("release data sync task: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return store.dataSyncTaskOwnershipError(ctx, taskID)
	}
	return nil
}

func (store *Store) ReleaseDataSyncTaskAfterSkippedFetch(ctx context.Context, taskID string, workerID string) error {
	commandTag, err := store.pool.Exec(ctx, fmt.Sprintf(`
				UPDATE data_sync_tasks
			   SET %s,
			       attempt_count = GREATEST(attempt_count - 1, 0),
			       updated_at = now()
			 WHERE id = $1
			   AND deleted_at IS NULL
			   AND locked_by = $2`,
		clearLeaseAssignments(dataSyncTaskLease),
	), taskID, workerID)
	if err != nil {
		return fmt.Errorf("release data sync task after skipped fetch: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return store.dataSyncTaskOwnershipError(ctx, taskID)
	}
	return nil
}

func dataSyncOwnedActiveLeaseWhere(taskIDArg string, workerIDArg string) string {
	return fmt.Sprintf(
		"id = %s AND deleted_at IS NULL AND status = 'running' AND locked_by = %s AND locked_until IS NOT NULL AND locked_until > now()",
		taskIDArg,
		workerIDArg,
	)
}

func (store *Store) dataSyncTaskOwnershipError(ctx context.Context, taskID string) error {
	if exists, err := store.dataSyncTaskExists(ctx, taskID); err != nil {
		return err
	} else if !exists {
		return data.ErrNotFound
	}
	return data.DataSyncCommandInvalidStateError()
}

func normalizeTaskError(taskErr error) string {
	return errtext.ExternalError(taskErr.Error())
}

func intervalLiteral(duration time.Duration) string {
	return fmt.Sprintf("%f seconds", duration.Seconds())
}
