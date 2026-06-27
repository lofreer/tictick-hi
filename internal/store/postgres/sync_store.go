package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
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

	var id string
	err = tx.QueryRow(ctx, `
		SELECT id
		  FROM data_sync_tasks
		 WHERE (sync_enabled = true OR realtime_enabled = true)
		   AND status <> $1
		   AND (locked_until IS NULL OR locked_until < now())
		 ORDER BY realtime_enabled DESC, created_at ASC
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		data.TaskStatusCancelled,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return data.DataSyncTask{}, false, nil
	}
	if err != nil {
		return data.DataSyncTask{}, false, fmt.Errorf("select data sync task: %w", err)
	}

	row := tx.QueryRow(ctx, `
		UPDATE data_sync_tasks
		   SET status = $2,
		       locked_by = $3,
		       locked_until = now() + $4::interval,
		       heartbeat_at = now(),
		       started_at = COALESCE(started_at, now()),
		       attempt_count = attempt_count + 1,
		       updated_at = now()
		 WHERE id = $1
		RETURNING id, exchange, symbol, interval, start_time, end_time,
		          sync_enabled, realtime_enabled, status, last_synced_open_time,
		          COALESCE(last_error, ''), attempt_count, created_at, updated_at`,
		id, data.TaskStatusRunning, workerID, intervalLiteral(leaseTTL),
	)
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
	commandTag, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET heartbeat_at = now(),
		       locked_until = now() + $3::interval,
		       updated_at = now()
		 WHERE id = $1
		   AND locked_by = $2
		   AND status = $4`,
		taskID,
		workerID,
		intervalLiteral(leaseTTL),
		data.TaskStatusRunning,
	)
	if err != nil {
		return fmt.Errorf("heartbeat data sync task: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
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

	if _, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE data_sync_tasks
		   SET last_synced_open_time = COALESCE($2, last_synced_open_time),
		       status = CASE
		         WHEN realtime_enabled THEN $3
		         WHEN $4::boolean THEN $5
		         ELSE $6
		       END,
		       sync_enabled = CASE
		         WHEN $4::boolean AND NOT realtime_enabled THEN false
		         ELSE sync_enabled
		       END,
		       %s,
		       finished_at = CASE
		         WHEN $4::boolean AND NOT realtime_enabled THEN now()
		         ELSE finished_at
		       END,
		       last_error = NULL,
		       updated_at = now()
		 WHERE id = $1`, clearLeaseAssignments(dataSyncTaskLease)),
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
	retryable := exchange.IsTemporaryError(taskErr)
	_, err := store.pool.Exec(ctx, fmt.Sprintf(`
		UPDATE data_sync_tasks
		   SET status = CASE
		         WHEN $4::boolean AND (sync_enabled OR realtime_enabled) THEN $5
		         ELSE $2
		       END,
		       %s,
		       last_error = $3,
		       updated_at = now()
		 WHERE id = $1`, clearLeaseAssignments(dataSyncTaskLease)),
		taskID,
		data.TaskStatusFailed,
		normalizeTaskError(taskErr),
		retryable,
		data.TaskStatusPending,
	)
	if err != nil {
		return fmt.Errorf("mark data sync failed: %w", err)
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
