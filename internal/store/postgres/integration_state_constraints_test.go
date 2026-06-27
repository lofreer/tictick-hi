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

func TestIntegrationTaskStatusTransitionGuards(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	dataPendingID := "dst_bad_transition_" + suffix
	dataFailedID := "dst_retry_transition_" + suffix
	backtestFailedID := "bt_bad_transition_" + suffix
	tradingFailedID := "tt_bad_transition_" + suffix
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id IN ($1, $2)`, dataPendingID, dataFailedID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM backtest_tasks WHERE id = $1`, backtestFailedID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM trading_tasks WHERE id = $1`, tradingFailedID)
	})

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (id, exchange, symbol, interval, status)
		VALUES ($1, 'binance', $2, '1m', 'pending')`,
		dataPendingID,
		"ITBADTRANSITION"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	_, err := store.pool.Exec(ctx, `UPDATE data_sync_tasks SET status = 'succeeded' WHERE id = $1`, dataPendingID)
	assertTransitionRejected(t, err, "data_sync_tasks_status_transition_check")

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, sync_enabled, realtime_enabled,
			status, finished_at
		)
		VALUES ($1, 'binance', $2, '1m', false, false, 'failed', now())`,
		dataFailedID,
		"ITRETRYTRANSITION"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		UPDATE data_sync_tasks
		   SET status = 'pending',
		       finished_at = NULL
		 WHERE id = $1`,
		dataFailedID,
	); err != nil {
		t.Fatalf("failed data sync task should be retryable: %v", err)
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO backtest_tasks (
			id, name, exchange, symbol, interval, strategy_id,
			initial_balance, trigger_mode, status, finished_at
		)
		VALUES ($1, 'bad transition', 'binance', $2, '1m',
		        'ema-cross', 10000, 'closed_candle', 'failed', now())`,
		backtestFailedID,
		"ITBADBTTRANSITION"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `UPDATE backtest_tasks SET status = 'pending' WHERE id = $1`, backtestFailedID)
	assertTransitionRejected(t, err, "backtest_tasks_status_transition_check")

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO trading_tasks (
			id, name, type, exchange, account_id, symbol, interval,
			strategy_id, status, finished_at
		)
		VALUES ($1, 'bad transition', 'paper', 'binance', 'paper',
		        $2, '1m', 'ema-cross', 'failed', now())`,
		tradingFailedID,
		"ITBADTTTRANSITION"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	_, err = store.pool.Exec(ctx, `UPDATE trading_tasks SET status = 'running' WHERE id = $1`, tradingFailedID)
	assertTransitionRejected(t, err, "trading_tasks_status_transition_check")
}

func TestIntegrationTaskCommandsRejectInvalidStatusTransitions(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	dataFailedID := "dst_command_transition_" + suffix
	tradingFailedID := "tt_command_transition_" + suffix
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id = $1`, dataFailedID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM trading_tasks WHERE id = $1`, tradingFailedID)
	})

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_tasks (
			id, exchange, symbol, interval, sync_enabled, realtime_enabled,
			status, finished_at
		)
		VALUES ($1, 'binance', $2, '1m', false, false, 'failed', now())`,
		dataFailedID,
		"ITDSTCOMMAND"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetSyncEnabled(ctx, dataFailedID, true); !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("set sync on failed task error = %v, want invalid state", err)
	}
	row := readIntegrationSyncTask(t, ctx, store, dataFailedID)
	if row.status != data.TaskStatusFailed {
		t.Fatalf("failed data sync task status changed to %s", row.status)
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO trading_tasks (
			id, name, type, exchange, account_id, symbol, interval,
			strategy_id, status, finished_at
		)
		VALUES ($1, 'bad command', 'paper', 'binance', 'paper',
		        $2, '1m', 'ema-cross', 'failed', now())`,
		tradingFailedID,
		"ITTT_COMMAND"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetTradingTaskStatus(ctx, tradingFailedID, data.TaskStatusRunning); !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("start failed trading task error = %v, want invalid state", err)
	}
	var tradingStatus data.TaskStatus
	if err := store.pool.QueryRow(ctx, `SELECT status FROM trading_tasks WHERE id = $1`, tradingFailedID).Scan(&tradingStatus); err != nil {
		t.Fatal(err)
	}
	if tradingStatus != data.TaskStatusFailed {
		t.Fatalf("failed trading task status changed to %s", tradingStatus)
	}
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

func assertTransitionRejected(t *testing.T, err error, constraint string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s violation", constraint)
	}
	if !strings.Contains(err.Error(), constraint) {
		t.Fatalf("error = %v, want constraint %s", err, constraint)
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
