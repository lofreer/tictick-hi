package postgres

import "testing"

func TestIntegrationTaskTrimmedRequiredTextConstraints(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	tests := []struct {
		name       string
		statement  string
		args       []any
		constraint string
	}{
		{
			name: "data sync task blank exchange",
			statement: `
				INSERT INTO data_sync_tasks (id, exchange, symbol, interval)
				VALUES ($1, '   ', 'BTCUSDT', '1m')`,
			args:       []any{integrationID("dst_blank_exchange")},
			constraint: "data_sync_tasks_trimmed_required_text_check",
		},
		{
			name: "backtest task blank name",
			statement: `
				INSERT INTO backtest_tasks (
					id, name, exchange, symbol, interval, strategy_id,
					initial_balance, fee_bps, slippage_bps
				)
				VALUES ($1, '   ', 'binance', 'BTCUSDT', '1m', 'ema-cross', 10000, 0, 0)`,
			args:       []any{integrationID("bt_blank_name")},
			constraint: "backtest_tasks_trimmed_required_text_check",
		},
		{
			name: "trading task blank name",
			statement: `
				INSERT INTO trading_tasks (
					id, name, type, exchange, account_id, symbol, interval, strategy_id
				)
				VALUES ($1, '   ', 'paper', 'binance', 'paper', 'BTCUSDT', '1m', 'ema-cross')`,
			args:       []any{integrationID("tt_blank_name")},
			constraint: "trading_tasks_trimmed_required_text_check",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := store.pool.Exec(ctx, test.statement, test.args...)
			if err == nil {
				t.Fatalf("expected %s violation", test.constraint)
			}
			assertDatabaseConstraintError(t, err, test.constraint)
		})
	}
}
