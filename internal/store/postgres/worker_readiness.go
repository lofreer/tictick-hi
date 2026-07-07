package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type WorkerQueueBacklogLimits struct {
	MaxBacklog  int
	MaxReadyAge time.Duration
}

func (store *Store) CheckWorkerQueue(ctx context.Context, command string) error {
	query, err := workerQueueReadinessQuery(command)
	if err != nil {
		return err
	}
	var count int64
	if err := store.pool.QueryRow(
		ctx,
		query,
		string(data.TaskStatusPending),
		string(data.TaskStatusRunning),
	).Scan(&count); err != nil {
		return fmt.Errorf("read %s worker queue: %w", command, err)
	}
	return nil
}

func (store *Store) CheckWorkerQueueBacklog(
	ctx context.Context,
	command string,
	limits WorkerQueueBacklogLimits,
) error {
	if limits.MaxBacklog <= 0 && limits.MaxReadyAge <= 0 {
		return nil
	}
	query, err := workerQueueBacklogReadinessQuery(command)
	if err != nil {
		return err
	}
	var count int
	var oldestReadyAt sql.NullTime
	if err := store.pool.QueryRow(ctx, query).Scan(&count, &oldestReadyAt); err != nil {
		return fmt.Errorf("read %s worker queue backlog: %w", command, err)
	}
	metrics := workerQueueBacklogMetrics{ReadyCount: count}
	if oldestReadyAt.Valid {
		readyAt := oldestReadyAt.Time
		metrics.OldestReadyAt = &readyAt
	}
	return checkWorkerQueueBacklogLimits(command, metrics, limits, time.Now().UTC())
}

func workerQueueReadinessQuery(command string) (string, error) {
	switch command {
	case "sync":
		return `
			SELECT count(*)
			  FROM data_sync_tasks
			 WHERE deleted_at IS NULL
			   AND status IN ($1, $2)`, nil
	case "backtest":
		return `
			SELECT count(*)
			  FROM backtest_tasks
			 WHERE status IN ($1, $2)`, nil
	case "trading":
		return `
			SELECT count(*)
			  FROM trading_tasks
			 WHERE status IN ($1, $2)`, nil
	case "notify":
		return `
			SELECT count(*)
			  FROM notification_outbox
			 WHERE status IN ($1, $2)`, nil
	default:
		return "", fmt.Errorf("unknown worker command %q", command)
	}
}

type workerQueueBacklogMetrics struct {
	ReadyCount    int
	OldestReadyAt *time.Time
}

func checkWorkerQueueBacklogLimits(
	command string,
	metrics workerQueueBacklogMetrics,
	limits WorkerQueueBacklogLimits,
	now time.Time,
) error {
	if limits.MaxBacklog > 0 && metrics.ReadyCount > limits.MaxBacklog {
		return fmt.Errorf(
			"%s worker ready backlog %d exceeds limit %d",
			command,
			metrics.ReadyCount,
			limits.MaxBacklog,
		)
	}
	if limits.MaxReadyAge > 0 && metrics.OldestReadyAt != nil {
		readyAge := now.Sub(metrics.OldestReadyAt.UTC())
		if readyAge > limits.MaxReadyAge {
			return fmt.Errorf(
				"%s worker oldest ready task age %s exceeds limit %s",
				command,
				readyAge.Round(time.Second),
				limits.MaxReadyAge,
			)
		}
	}
	return nil
}

func workerQueueBacklogReadinessQuery(command string) (string, error) {
	switch command {
	case "sync":
		return `
			SELECT count(*)::int,
			       min(COALESCE(next_attempt_at, created_at))
			  FROM data_sync_tasks
			 WHERE deleted_at IS NULL
			   AND status = 'pending'
			   AND (sync_enabled = true OR realtime_enabled = true)
			   AND (next_attempt_at IS NULL OR next_attempt_at <= now())
			   AND (locked_until IS NULL OR locked_until < now())
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
			   )`, nil
	case "backtest":
		return `
			SELECT count(*)::int,
			       min(created_at)
			  FROM backtest_tasks
			 WHERE status = 'pending'
			   AND (locked_until IS NULL OR locked_until < now())`, nil
	case "trading":
		return `
			SELECT count(*)::int,
			       min(updated_at)
			  FROM trading_tasks
			 WHERE status = 'running'
			   AND (locked_until IS NULL OR locked_until < now())`, nil
	case "notify":
		return `
			SELECT count(*)::int,
			       min(COALESCE(next_attempt_at, created_at))
			  FROM notification_outbox
			 WHERE status IN ('pending', 'retry_scheduled')
			   AND next_attempt_at <= now()
			   AND (locked_until IS NULL OR locked_until < now())`, nil
	default:
		return "", fmt.Errorf("unknown worker command %q", command)
	}
}
