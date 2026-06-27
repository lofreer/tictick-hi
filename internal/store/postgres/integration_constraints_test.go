package postgres

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestIntegrationDatabaseConstraintsRejectInvalidDomainValues(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	parentTaskID, _, parentOrderID, parentNotificationID := seedIntegrationTradingGraph(t, ctx, store, "domain_"+suffix)
	cases := []struct {
		name       string
		statement  string
		args       []any
		constraint string
	}{
		{
			name: "data sync status",
			statement: `
				INSERT INTO data_sync_tasks (id, exchange, symbol, interval, status)
				VALUES ($1, 'binance', $2, '1m', 'bogus')`,
			args:       []any{"dst_bad_status_" + suffix, "ITBADSTATUS" + suffix + "USDT"},
			constraint: "data_sync_tasks_status_check",
		},
		{
			name: "candle OHLC bounds",
			statement: `
				INSERT INTO market_candles (
					exchange, symbol, interval, open_time, close_time,
					open, high, low, close, volume, is_closed
				)
				VALUES ('binance', $1, '1m', $2, $3, 100, 90, 95, 100, 1, true)`,
			args: []any{
				"ITBADOHLC" + suffix + "USDT",
				time.Date(2026, 6, 27, 3, 0, 0, 0, time.UTC),
				time.Date(2026, 6, 27, 3, 1, 0, 0, time.UTC),
			},
			constraint: "market_candles_ohlc_bounds_check",
		},
		{
			name: "backtest trigger mode",
			statement: `
				INSERT INTO backtest_tasks (
					id, name, exchange, symbol, interval, strategy_id,
					initial_balance, trigger_mode
				)
				VALUES ($1, 'bad trigger', 'binance', $2, '1m', 'ema-cross', 10000, 'tick')`,
			args:       []any{"bt_bad_trigger_" + suffix, "ITBADTRIGGER" + suffix + "USDT"},
			constraint: "backtest_tasks_trigger_mode_check",
		},
		{
			name: "trading task type",
			statement: `
				INSERT INTO trading_tasks (
					id, name, type, exchange, account_id, symbol, strategy_id
				)
				VALUES ($1, 'bad type', 'demo', 'binance', 'paper', $2, 'ema-cross')`,
			args:       []any{"tt_bad_type_" + suffix, "ITBADTYPE" + suffix + "USDT"},
			constraint: "trading_tasks_type_check",
		},
		{
			name: "strategy intent type",
			statement: `
				INSERT INTO strategy_intents (
					id, task_id, task_type, strategy_id, intent_type,
					idempotency_key, policy, status
				)
				VALUES ($1, 'task', 'paper', 'ema-cross', 'email', $2, 'notify', 'accepted')`,
			args:       []any{"si_bad_type_" + suffix, "intent_bad_type_" + suffix},
			constraint: "strategy_intents_intent_type_check",
		},
		{
			name: "order side",
			statement: `
				INSERT INTO orders (
					id, task_id, task_type, idempotency_key, exchange, account_id,
					symbol, side, order_type, price, quantity, status
				)
				VALUES ($1, $2, 'paper', $3, 'binance', 'paper', $4, 'hold',
				        'market', 100, 1, 'filled')`,
			args:       []any{"ord_bad_side_" + suffix, parentTaskID, "order_bad_side_" + suffix, "ITBADORDER" + suffix + "USDT"},
			constraint: "orders_side_check",
		},
		{
			name: "notification provider",
			statement: `
				INSERT INTO notification_channels (id, name, provider, target)
				VALUES ($1, $2, 'smtp', 'demo')`,
			args:       []any{"nc_bad_provider_" + suffix, "bad-provider-" + suffix},
			constraint: "notification_channels_provider_check",
		},
		{
			name: "notification attempt count",
			statement: `
				INSERT INTO notifications (
					id, task_id, channel, provider, target, title, body,
					status, attempt_count, max_attempts
				)
				VALUES ($1, $2, 'default', 'local', 'default', 'title', 'body',
				        'failed', -1, 3)`,
			args:       []any{"nt_bad_attempt_" + suffix, parentTaskID},
			constraint: "notifications_attempt_bounds_check",
		},
		{
			name: "execution quantity",
			statement: `
				INSERT INTO executions (
					id, task_id, task_type, order_id, idempotency_key, exchange,
					account_id, symbol, side, price, quantity, status, executed_at
				)
				VALUES ($1, $2, 'paper', $3, $4, 'binance', 'paper',
				        $5, 'buy', 100, 0, 'filled', $6)`,
			args: []any{
				"exe_bad_quantity_" + suffix,
				parentTaskID,
				parentOrderID,
				"execution_bad_quantity_" + suffix,
				"ITBADEXEC" + suffix + "USDT",
				time.Date(2026, 6, 27, 3, 2, 0, 0, time.UTC),
			},
			constraint: "executions_decimal_bounds_check",
		},
		{
			name: "data sync partial lease",
			statement: `
				INSERT INTO data_sync_tasks (
					id, exchange, symbol, interval, status, locked_by
				)
				VALUES ($1, 'binance', $2, '1m', 'running', 'worker')`,
			args:       []any{"dst_bad_lease_" + suffix, "ITBADLEASE" + suffix + "USDT"},
			constraint: "data_sync_tasks_lease_consistency_check",
		},
		{
			name: "backtest non running lease",
			statement: `
				INSERT INTO backtest_tasks (
					id, name, exchange, symbol, interval, strategy_id,
					initial_balance, trigger_mode, status,
					locked_by, locked_until, heartbeat_at
				)
				VALUES ($1, 'bad lease', 'binance', $2, '1m', 'ema-cross',
				        10000, 'closed_candle', 'pending', 'worker', $3, $3)`,
			args: []any{
				"bt_bad_lease_" + suffix,
				"ITBADBTLEASE" + suffix + "USDT",
				time.Date(2026, 6, 27, 3, 3, 0, 0, time.UTC),
			},
			constraint: "backtest_tasks_lease_consistency_check",
		},
		{
			name: "trading missing heartbeat lease",
			statement: `
				INSERT INTO trading_tasks (
					id, name, type, exchange, account_id, symbol, strategy_id,
					status, locked_by, locked_until
				)
				VALUES ($1, 'bad lease', 'paper', 'binance', 'paper',
				        $2, 'ema-cross', 'running', 'worker', $3)`,
			args: []any{
				"tt_bad_lease_" + suffix,
				"ITBADTTLEASE" + suffix + "USDT",
				time.Date(2026, 6, 27, 3, 4, 0, 0, time.UTC),
			},
			constraint: "trading_tasks_lease_consistency_check",
		},
		{
			name: "notification outbox non running lease",
			statement: `
				INSERT INTO notification_outbox (
					id, notification_id, task_id, channel, provider, target,
					title, body, status, locked_by, locked_until
				)
				VALUES ($1, $2, $3, 'default', 'local', 'default',
				        'title', 'body', 'pending', 'worker', $4)`,
			args: []any{
				"no_bad_lease_" + suffix,
				parentNotificationID,
				parentTaskID,
				time.Date(2026, 6, 27, 3, 5, 0, 0, time.UTC),
			},
			constraint: "notification_outbox_lease_consistency_check",
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

func TestIntegrationNotificationProviderConstraintsAllowExternalProviders(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	providers := map[string]string{
		"email":    "smtp://smtp.example.com:587?from=bot@example.com&to=ops@example.com&username_env=SMTP_USERNAME&password_env=SMTP_PASSWORD",
		"telegram": "telegram://send?chat_id=ops-chat&token_env=TELEGRAM_BOT_TOKEN",
		"feishu":   "feishu://webhook?url_env=FEISHU_WEBHOOK_URL",
	}
	for provider, target := range providers {
		id := "nc_" + provider + "_" + suffix
		if _, err := store.pool.Exec(ctx, `
			INSERT INTO notification_channels (id, name, provider, target)
			VALUES ($1, $2, $3, $4)`,
			id,
			provider+"-"+suffix,
			provider,
			target,
		); err != nil {
			t.Fatalf("insert %s notification channel: %v", provider, err)
		}
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM notification_channels WHERE name LIKE $1`, "%"+suffix)
	})
}

func TestIntegrationDatabaseReferentialConstraintsRejectOrphans(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	suffix := fmt.Sprintf("%d", time.Now().UTC().UnixNano())
	parentTaskID, parentIntentID, parentOrderID, parentNotificationID := seedIntegrationTradingGraph(t, ctx, store, "fk_"+suffix)
	backtestTaskID := "bt_fk_parent_" + suffix
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO backtest_tasks (
			id, name, exchange, symbol, interval, strategy_id,
			initial_balance, trigger_mode
		)
		VALUES ($1, 'fk parent', 'binance', $2, '1m', 'ema-cross', 10000, 'closed_candle')`,
		backtestTaskID,
		"ITFKBT"+suffix+"USDT",
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM strategy_intents WHERE task_id = $1`, backtestTaskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM backtest_tasks WHERE id = $1`, backtestTaskID)
	})
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO strategy_intents (
			id, task_id, task_type, strategy_id, intent_type,
			idempotency_key, policy, status
		)
		VALUES ($1, $2, 'backtest', 'ema-cross', 'order', $3, 'simulate', 'accepted')`,
		"si_valid_backtest_"+suffix,
		backtestTaskID,
		"intent_valid_backtest_"+suffix,
	); err != nil {
		t.Fatalf("insert valid backtest intent: %v", err)
	}

	cases := []struct {
		name       string
		statement  string
		args       []any
		constraint string
	}{
		{
			name: "order missing trading task",
			statement: `
				INSERT INTO orders (
					id, task_id, task_type, idempotency_key, exchange, account_id,
					symbol, side, order_type, price, quantity, status
				)
				VALUES ($1, $2, 'paper', $3, 'binance', 'paper', $4, 'buy',
				        'market', 100, 1, 'filled')`,
			args: []any{
				"ord_fk_missing_task_" + suffix,
				"tt_missing_" + suffix,
				"ord_fk_missing_task_key_" + suffix,
				"ITFKORDER" + suffix + "USDT",
			},
			constraint: "orders_trading_task_fk",
		},
		{
			name: "strategy intent missing backtest task",
			statement: `
				INSERT INTO strategy_intents (
					id, task_id, task_type, strategy_id, intent_type,
					idempotency_key, policy, status
				)
				VALUES ($1, $2, 'backtest', 'ema-cross', 'order', $3,
				        'simulate', 'accepted')`,
			args: []any{
				"si_fk_missing_backtest_" + suffix,
				"bt_missing_" + suffix,
				"si_fk_missing_backtest_key_" + suffix,
			},
			constraint: "strategy_intents_task_parent_fk",
		},
		{
			name: "strategy intent missing trading task",
			statement: `
				INSERT INTO strategy_intents (
					id, task_id, task_type, strategy_id, intent_type,
					idempotency_key, policy, status
				)
				VALUES ($1, $2, 'paper', 'ema-cross', 'order', $3,
				        'execute', 'accepted')`,
			args: []any{
				"si_fk_missing_trading_" + suffix,
				"tt_missing_" + suffix,
				"si_fk_missing_trading_key_" + suffix,
			},
			constraint: "strategy_intents_task_parent_fk",
		},
		{
			name: "strategy intent trading type mismatch",
			statement: `
				INSERT INTO strategy_intents (
					id, task_id, task_type, strategy_id, intent_type,
					idempotency_key, policy, status
				)
				VALUES ($1, $2, 'live', 'ema-cross', 'order', $3,
				        'execute', 'accepted')`,
			args: []any{
				"si_fk_mismatch_trading_" + suffix,
				parentTaskID,
				"si_fk_mismatch_trading_key_" + suffix,
			},
			constraint: "strategy_intents_task_parent_fk",
		},
		{
			name: "order missing intent",
			statement: `
				INSERT INTO orders (
					id, task_id, task_type, intent_id, idempotency_key, exchange,
					account_id, symbol, side, order_type, price, quantity, status
				)
				VALUES ($1, $2, 'paper', $3, $4, 'binance', 'paper', $5, 'buy',
				        'market', 100, 1, 'filled')`,
			args: []any{
				"ord_fk_missing_intent_" + suffix,
				parentTaskID,
				"si_missing_" + suffix,
				"ord_fk_missing_intent_key_" + suffix,
				"ITFKORDERINTENT" + suffix + "USDT",
			},
			constraint: "orders_intent_task_fk",
		},
		{
			name: "execution missing order",
			statement: `
				INSERT INTO executions (
					id, task_id, task_type, order_id, intent_id, idempotency_key,
					exchange, account_id, symbol, side, price, quantity, status,
					executed_at
				)
				VALUES ($1, $2, 'paper', $3, $4, $5, 'binance', 'paper', $6,
				        'buy', 100, 1, 'filled', $7)`,
			args: []any{
				"exe_fk_missing_order_" + suffix,
				parentTaskID,
				"ord_missing_" + suffix,
				parentIntentID,
				"exe_fk_missing_order_key_" + suffix,
				"ITFKEXECORDER" + suffix + "USDT",
				time.Date(2026, 6, 27, 4, 0, 0, 0, time.UTC),
			},
			constraint: "executions_order_task_fk",
		},
		{
			name: "execution missing intent",
			statement: `
				INSERT INTO executions (
					id, task_id, task_type, order_id, intent_id, idempotency_key,
					exchange, account_id, symbol, side, price, quantity, status,
					executed_at
				)
				VALUES ($1, $2, 'paper', $3, $4, $5, 'binance', 'paper', $6,
				        'buy', 100, 1, 'filled', $7)`,
			args: []any{
				"exe_fk_missing_intent_" + suffix,
				parentTaskID,
				parentOrderID,
				"si_missing_" + suffix,
				"exe_fk_missing_intent_key_" + suffix,
				"ITFKEXECINTENT" + suffix + "USDT",
				time.Date(2026, 6, 27, 4, 1, 0, 0, time.UTC),
			},
			constraint: "executions_intent_task_fk",
		},
		{
			name: "position missing trading task",
			statement: `
				INSERT INTO positions (
					task_id, task_type, exchange, account_id, symbol,
					quantity, average_price, realized_pnl
				)
				VALUES ($1, 'paper', 'binance', 'paper', $2, 1, 100, 0)`,
			args:       []any{"tt_missing_position_" + suffix, "ITFKPOSITION" + suffix + "USDT"},
			constraint: "positions_trading_task_fk",
		},
		{
			name: "notification missing trading task",
			statement: `
				INSERT INTO notifications (
					id, task_id, channel, provider, target, title, body, status
				)
				VALUES ($1, $2, 'default', 'local', 'default', 'title', 'body', 'pending')`,
			args:       []any{"nt_fk_missing_task_" + suffix, "tt_missing_notification_" + suffix},
			constraint: "notifications_trading_task_fk",
		},
		{
			name: "notification missing intent",
			statement: `
				INSERT INTO notifications (
					id, task_id, intent_id, channel, provider, target, title, body, status
				)
				VALUES ($1, $2, $3, 'default', 'local', 'default', 'title', 'body', 'pending')`,
			args:       []any{"nt_fk_missing_intent_" + suffix, parentTaskID, "si_missing_" + suffix},
			constraint: "notifications_intent_task_fk",
		},
		{
			name: "notification outbox missing intent",
			statement: `
				INSERT INTO notification_outbox (
					id, notification_id, task_id, intent_id, channel, provider,
					target, title, body, status, next_attempt_at
				)
				VALUES ($1, $2, $3, $4, 'default', 'local', 'default',
				        'title', 'body', 'pending', $5)`,
			args: []any{
				"no_fk_missing_intent_" + suffix,
				parentNotificationID,
				parentTaskID,
				"si_missing_" + suffix,
				time.Date(2026, 6, 27, 4, 2, 0, 0, time.UTC),
			},
			constraint: "notification_outbox_intent_task_fk",
		},
		{
			name: "backtest order missing intent",
			statement: `
				INSERT INTO backtest_orders (
					id, backtest_id, intent_id, side, price, quantity, status, occurred_at
				)
				VALUES ($1, $2, $3, 'buy', 100, 1, 'filled', $4)`,
			args: []any{
				"bo_fk_missing_intent_" + suffix,
				backtestTaskID,
				"si_missing_" + suffix,
				time.Date(2026, 6, 27, 4, 3, 0, 0, time.UTC),
			},
			constraint: "backtest_orders_intent_task_fk",
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

func seedIntegrationTradingGraph(
	t *testing.T,
	ctx context.Context,
	store *Store,
	suffix string,
) (taskID string, intentID string, orderID string, notificationID string) {
	t.Helper()

	taskID = "tt_parent_" + suffix
	intentID = "si_parent_" + suffix
	orderID = "ord_parent_" + suffix
	notificationID = "nt_parent_" + suffix
	symbol := "ITPARENT" + suffix + "USDT"

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO trading_tasks (
			id, name, type, exchange, account_id, symbol, interval, strategy_id
		)
		VALUES ($1, 'constraint parent', 'paper', 'binance', 'paper', $2, '1m', 'ema-cross')`,
		taskID,
		symbol,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO strategy_intents (
			id, task_id, task_type, strategy_id, intent_type,
			idempotency_key, policy, status
		)
		VALUES ($1, $2, 'paper', 'ema-cross', 'order', $3, 'execute', 'executed')`,
		intentID,
		taskID,
		"intent_parent_"+suffix,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO orders (
			id, task_id, task_type, intent_id, idempotency_key, exchange,
			account_id, symbol, side, order_type, price, quantity, status
		)
		VALUES ($1, $2, 'paper', $3, $4, 'binance', 'paper', $5, 'buy',
		        'market', 100, 1, 'filled')`,
		orderID,
		taskID,
		intentID,
		"order_parent_"+suffix,
		symbol,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO notifications (
			id, task_id, intent_id, channel, provider, target, title, body, status
		)
		VALUES ($1, $2, $3, 'default', 'local', 'default', 'title', 'body', 'pending')`,
		notificationID,
		taskID,
		intentID,
	); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM trading_tasks WHERE id = $1`, taskID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM strategy_intents WHERE task_id = $1`, taskID)
	})

	return taskID, intentID, orderID, notificationID
}
