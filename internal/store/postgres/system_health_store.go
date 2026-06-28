package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) SystemHealth(ctx context.Context) (data.SystemHealth, error) {
	checkedAt := time.Now().UTC()
	if err := store.pool.Ping(ctx); err != nil {
		return data.SystemHealth{
			Status:    "degraded",
			Database:  "failed",
			CheckedAt: checkedAt,
			Services: []data.ServiceHealth{
				{Name: "postgres", Status: "failed", Detail: err.Error()},
				{Name: "api", Status: "ok"},
			},
		}, nil
	}
	services := []data.ServiceHealth{
		{Name: "postgres", Status: "ok"},
		{Name: "api", Status: "ok"},
	}
	workerChecks := []struct {
		name  string
		query string
	}{
		{
			name: "sync-worker",
			query: `
				SELECT count(*) FILTER (WHERE status = 'pending')::int,
				       count(*) FILTER (WHERE status = 'running')::int,
				       count(*) FILTER (WHERE locked_by IS NOT NULL AND locked_until IS NOT NULL)::int,
				       count(*) FILTER (WHERE locked_until IS NOT NULL AND locked_until < now())::int,
				       max(heartbeat_at),
				       max(locked_until)
				  FROM data_sync_tasks`,
		},
		{
			name: "backtest-worker",
			query: `
				SELECT count(*) FILTER (WHERE status = 'pending')::int,
				       count(*) FILTER (WHERE status = 'running')::int,
				       count(*) FILTER (WHERE locked_by IS NOT NULL AND locked_until IS NOT NULL)::int,
				       count(*) FILTER (WHERE locked_until IS NOT NULL AND locked_until < now())::int,
				       max(heartbeat_at),
				       max(locked_until)
				  FROM backtest_tasks`,
		},
		{
			name: "trading-worker",
			query: `
				SELECT count(*) FILTER (WHERE status = 'running')::int,
				       count(*) FILTER (WHERE status = 'running')::int,
				       count(*) FILTER (WHERE locked_by IS NOT NULL AND locked_until IS NOT NULL)::int,
				       count(*) FILTER (WHERE locked_until IS NOT NULL AND locked_until < now())::int,
				       max(heartbeat_at),
				       max(locked_until)
				  FROM trading_tasks`,
		},
		{
			name: "notify-worker",
			query: `
				SELECT count(*) FILTER (WHERE status IN ('pending', 'retry_scheduled'))::int,
				       count(*) FILTER (WHERE status = 'running')::int,
				       count(*) FILTER (WHERE locked_by IS NOT NULL AND locked_until IS NOT NULL)::int,
				       count(*) FILTER (WHERE locked_until IS NOT NULL AND locked_until < now())::int,
				       max(last_attempt_at),
				       max(locked_until)
				  FROM notification_outbox`,
		},
	}
	status := "ok"
	for _, check := range workerChecks {
		service, err := store.workerHealth(ctx, check.name, check.query)
		if err != nil {
			return data.SystemHealth{}, err
		}
		if check.name == "sync-worker" {
			if err := store.addSyncExchangeBackoffHealth(ctx, &service); err != nil {
				return data.SystemHealth{}, err
			}
		}
		if service.Status != "ok" {
			status = "degraded"
		}
		services = append(services, service)
	}
	return data.SystemHealth{
		Status:    status,
		Database:  "ok",
		CheckedAt: checkedAt,
		Services:  services,
	}, nil
}

func (store *Store) workerHealth(ctx context.Context, name string, query string) (data.ServiceHealth, error) {
	var pendingCount int
	var runningCount int
	var lockedCount int
	var staleLeaseCount int
	var heartbeat sql.NullTime
	var lockedUntil sql.NullTime
	service := data.ServiceHealth{Name: name}
	err := store.pool.QueryRow(ctx, query).Scan(
		&pendingCount,
		&runningCount,
		&lockedCount,
		&staleLeaseCount,
		&heartbeat,
		&lockedUntil,
	)
	if err != nil {
		return data.ServiceHealth{}, fmt.Errorf("read %s health: %w", name, err)
	}
	service.Status = "ok"
	if staleLeaseCount > 0 {
		service.Status = "warning"
	}
	service.PendingCount = &pendingCount
	service.RunningCount = &runningCount
	service.LockedCount = &lockedCount
	service.StaleLeaseCount = &staleLeaseCount
	service.Detail = fmt.Sprintf(
		"pending=%d running=%d locked=%d stale=%d",
		pendingCount,
		runningCount,
		lockedCount,
		staleLeaseCount,
	)
	if heartbeat.Valid {
		service.LastHeartbeatAt = &heartbeat.Time
	}
	if lockedUntil.Valid {
		service.LockedUntil = &lockedUntil.Time
	}
	return service, nil
}

func (store *Store) addSyncExchangeBackoffHealth(ctx context.Context, service *data.ServiceHealth) error {
	var backoffCount int
	var nextAttempt sql.NullTime
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE next_attempt_at > now())::int,
		       min(next_attempt_at) FILTER (WHERE next_attempt_at > now())
		  FROM data_sync_exchange_backoffs`,
	).Scan(&backoffCount, &nextAttempt); err != nil {
		return fmt.Errorf("read sync exchange backoff health: %w", err)
	}

	service.ExchangeBackoffCount = &backoffCount
	if nextAttempt.Valid {
		service.NextExchangeAttemptAt = &nextAttempt.Time
	}
	if backoffCount > 0 {
		service.Status = "warning"
		service.Detail = fmt.Sprintf("%s exchange_backoff=%d", service.Detail, backoffCount)
	}
	return nil
}
