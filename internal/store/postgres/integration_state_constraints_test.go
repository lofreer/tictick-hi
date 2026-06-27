package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationTaskTerminalStatusesRequireFinishedAt(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	cases := []struct {
		name       string
		statement  string
		args       []any
		constraint string
	}{
		{
			name: "data sync",
			statement: `
				INSERT INTO data_sync_tasks (id, exchange, symbol, interval, status)
				VALUES ($1, 'binance', $2, '1m', 'succeeded')`,
			args:       []any{"dst_bad_terminal_" + suffix, "ITBADTERMINAL" + suffix + "USDT"},
			constraint: "data_sync_tasks_terminal_finished_at_check",
		},
		{
			name: "backtest",
			statement: `
				INSERT INTO backtest_tasks (
					id, name, exchange, symbol, interval, strategy_id,
					initial_balance, trigger_mode, status
				)
				VALUES ($1, 'bad terminal', 'binance', $2, '1m',
				        'ema-cross', 10000, 'closed_candle', 'failed')`,
			args:       []any{"bt_bad_terminal_" + suffix, "ITBADBTTERM" + suffix + "USDT"},
			constraint: "backtest_tasks_terminal_finished_at_check",
		},
		{
			name: "trading",
			statement: `
				INSERT INTO trading_tasks (
					id, name, type, exchange, account_id, symbol, strategy_id,
					status
				)
				VALUES ($1, 'bad terminal', 'paper', 'binance', 'paper',
				        $2, 'ema-cross', 'cancelled')`,
			args:       []any{"tt_bad_terminal_" + suffix, "ITBADTTTERM" + suffix + "USDT"},
			constraint: "trading_tasks_terminal_finished_at_check",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, err := store.pool.Exec(ctx, testCase.statement, testCase.args...)
			if err == nil {
				t.Fatalf("expected %s violation", testCase.constraint)
			}
			if !strings.Contains(err.Error(), testCase.constraint) {
				t.Fatalf("error = %v, want constraint %s", err, testCase.constraint)
			}
		})
	}
}

func TestIntegrationWorkerLeaseConstraintsAreValidated(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	constraints := []string{
		"data_sync_tasks_lease_consistency_check",
		"backtest_tasks_lease_consistency_check",
		"trading_tasks_lease_consistency_check",
		"notification_outbox_lease_consistency_check",
	}
	for _, constraint := range constraints {
		t.Run(constraint, func(t *testing.T) {
			var validated bool
			if err := store.pool.QueryRow(ctx, `
				SELECT COALESCE(
				  (
				    SELECT convalidated
				      FROM pg_constraint
				     WHERE conname = $1
				  ),
				  false
				)`,
				constraint,
			).Scan(&validated); err != nil {
				t.Fatal(err)
			}
			if !validated {
				t.Fatalf("%s is not validated", constraint)
			}
		})
	}
}

func TestIntegrationFailureTransitionsSetFinishedAt(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	syncTaskID := integrationID("dst")
	syncSymbol := integrationSymbol("TF")
	insertIntegrationSyncTask(t, ctx, store, syncTaskID, syncSymbol, data.TaskStatusRunning, true, true, "failed-worker")
	tradingTaskID := integrationID("tt")
	tradingSymbol := integrationSymbol("TF")
	insertRunningIntegrationTradingTask(t, ctx, store, tradingTaskID, tradingSymbol)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, syncTaskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM trading_tasks WHERE id = $1`, tradingTaskID)
	})

	if err := store.MarkDataSyncFailed(ctx, syncTaskID, errors.New("invalid symbol")); err != nil {
		t.Fatal(err)
	}
	assertFinishedAtExists(ctx, t, store, "data_sync_tasks", syncTaskID)

	if err := store.MarkTradingTaskFailed(ctx, tradingTaskID, errors.New("risk limit")); err != nil {
		t.Fatal(err)
	}
	assertFinishedAtExists(ctx, t, store, "trading_tasks", tradingTaskID)
}

func insertRunningIntegrationTradingTask(
	t *testing.T,
	ctx context.Context,
	store *Store,
	id string,
	symbol string,
) {
	t.Helper()
	now := time.Now().UTC()
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO trading_tasks (
			id, name, type, exchange, account_id, symbol, interval,
			strategy_id, status, locked_by, locked_until, heartbeat_at
		)
		VALUES ($1, 'terminal transition', 'paper', 'binance', 'paper',
		        $2, '1m', 'ema-cross', 'running', 'failed-worker', $3, $3)`,
		id,
		symbol,
		now.Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
}

func assertFinishedAtExists(ctx context.Context, t *testing.T, store *Store, table string, id string) {
	t.Helper()
	var finishedAt sql.NullTime
	if err := store.pool.QueryRow(ctx, `SELECT finished_at FROM `+table+` WHERE id = $1`, id).Scan(&finishedAt); err != nil {
		t.Fatal(err)
	}
	if !finishedAt.Valid {
		t.Fatalf("%s %s finished_at is null", table, id)
	}
}
