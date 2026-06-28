package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
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
				   AND status IN ($1, $2)
				   AND (next_attempt_at IS NULL OR next_attempt_at <= now())`,
			orderBy: "realtime_enabled DESC, created_at ASC",
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
			returningColumns: `id, exchange, symbol, interval, start_time, end_time,
		          sync_enabled, realtime_enabled, status, last_synced_open_time,
		          COALESCE(last_error, ''), attempt_count, next_attempt_at, created_at, updated_at`,
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
		return fmt.Errorf("heartbeat data sync task: lease lost for %s", taskID)
	}
	return nil
}

func (store *Store) SaveDataSyncResult(ctx context.Context, result data.DataSyncResult) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin save sync result: %w", err)
	}
	defer tx.Rollback(ctx)

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
		where: "id = $1",
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

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sync result: %w", err)
	}
	return nil
}

func (store *Store) MarkDataSyncFailed(ctx context.Context, taskID string, taskErr error) error {
	_, err := store.pool.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: dataSyncTaskLease,
		assignments: []string{
			"status = $2",
			"sync_enabled = false",
			"realtime_enabled = false",
			"last_error = $3",
			"next_attempt_at = NULL",
			"finished_at = now()",
		},
		where: "id = $1",
	}),
		taskID,
		data.TaskStatusFailed,
		normalizeTaskError(taskErr),
	)
	if err != nil {
		return fmt.Errorf("mark data sync failed: %w", err)
	}
	return nil
}

func (store *Store) RecordDataSyncRetry(
	ctx context.Context,
	taskID string,
	taskErr error,
	nextAttemptAt *time.Time,
) error {
	_, err := store.pool.Exec(ctx, leaseTransitionUpdateSQL(leaseTransitionUpdate{
		resource: dataSyncTaskLease,
		assignments: []string{
			`status = CASE
			         WHEN realtime_enabled THEN $2
			         ELSE $3
			       END`,
			"last_error = $4",
			"next_attempt_at = $5",
		},
		where: "id = $1",
	}),
		taskID,
		data.TaskStatusRunning,
		data.TaskStatusPending,
		normalizeTaskError(taskErr),
		nextAttemptAt,
	)
	if err != nil {
		return fmt.Errorf("record data sync retry: %w", err)
	}
	return nil
}

func (store *Store) ReleaseDataSyncTask(ctx context.Context, taskID string) error {
	if err := releaseLease(ctx, store.pool, dataSyncTaskLease, taskID); err != nil {
		return fmt.Errorf("release data sync task: %w", err)
	}
	return nil
}

func normalizeTaskError(taskErr error) string {
	message := strings.Join(strings.Fields(taskErr.Error()), " ")
	const maxRunes = 500
	runes := []rune(message)
	if len(runes) <= maxRunes {
		return message
	}
	return string(runes[:maxRunes-3]) + "..."
}

func intervalLiteral(duration time.Duration) string {
	return fmt.Sprintf("%f seconds", duration.Seconds())
}
