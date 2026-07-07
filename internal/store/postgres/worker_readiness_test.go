package postgres

import (
	"strings"
	"testing"
	"time"
)

func TestCheckWorkerQueueBacklogLimits(t *testing.T) {
	now := time.Date(2026, 7, 7, 10, 0, 0, 0, time.UTC)
	oldest := now.Add(-2 * time.Minute)

	if err := checkWorkerQueueBacklogLimits("sync", workerQueueBacklogMetrics{
		ReadyCount:    3,
		OldestReadyAt: &oldest,
	}, WorkerQueueBacklogLimits{MaxBacklog: 3, MaxReadyAge: 2 * time.Minute}, now); err != nil {
		t.Fatalf("exact limits should pass: %v", err)
	}

	err := checkWorkerQueueBacklogLimits("sync", workerQueueBacklogMetrics{
		ReadyCount:    4,
		OldestReadyAt: &oldest,
	}, WorkerQueueBacklogLimits{MaxBacklog: 3}, now)
	if err == nil || !strings.Contains(err.Error(), "ready backlog 4 exceeds limit 3") {
		t.Fatalf("expected backlog limit error, got %v", err)
	}

	err = checkWorkerQueueBacklogLimits("notify", workerQueueBacklogMetrics{
		ReadyCount:    1,
		OldestReadyAt: &oldest,
	}, WorkerQueueBacklogLimits{MaxReadyAge: time.Minute}, now)
	if err == nil || !strings.Contains(err.Error(), "oldest ready task age 2m0s exceeds limit 1m0s") {
		t.Fatalf("expected ready age limit error, got %v", err)
	}
}

func TestWorkerQueueBacklogReadinessQueries(t *testing.T) {
	tests := []struct {
		command string
		want    []string
		avoid   []string
	}{
		{
			command: "sync",
			want: []string{
				"FROM data_sync_tasks",
				"status = 'pending'",
				"next_attempt_at IS NULL OR next_attempt_at <= now()",
				"market_instruments",
				"data_sync_exchange_backoffs",
			},
			avoid: []string{"status = 'running'"},
		},
		{
			command: "backtest",
			want: []string{
				"FROM backtest_tasks",
				"status = 'pending'",
				"locked_until IS NULL OR locked_until < now()",
			},
		},
		{
			command: "trading",
			want: []string{
				"FROM trading_tasks",
				"status = 'running'",
				"min(updated_at)",
			},
		},
		{
			command: "notify",
			want: []string{
				"FROM notification_outbox",
				"status IN ('pending', 'retry_scheduled')",
				"next_attempt_at <= now()",
			},
			avoid: []string{"status = 'running'"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			query, err := workerQueueBacklogReadinessQuery(tt.command)
			if err != nil {
				t.Fatalf("query: %v", err)
			}
			for _, want := range tt.want {
				if !strings.Contains(query, want) {
					t.Fatalf("query missing %q:\n%s", want, query)
				}
			}
			for _, avoid := range tt.avoid {
				if strings.Contains(query, avoid) {
					t.Fatalf("query unexpectedly contains %q:\n%s", avoid, query)
				}
			}
		})
	}

	if _, err := workerQueueBacklogReadinessQuery("unknown"); err == nil {
		t.Fatal("expected unknown worker command error")
	}
}
