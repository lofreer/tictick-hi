package postgres

import (
	"context"
	"fmt"

	"github.com/lofreer/tictick-hi/internal/data"
)

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
