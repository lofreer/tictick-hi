package postgres

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationListOverviewRecentFactsReturnsGlobalFacts(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	now := time.Date(2026, 7, 7, 1, 0, 0, 0, time.UTC)
	backtest, err := store.CreateBacktestTask(ctx, data.CreateBacktestTask{
		Name:           integrationID("Overview Backtest"),
		Exchange:       "binance",
		Symbol:         integrationSymbol("OB"),
		Interval:       "1m",
		StrategyID:     "ema-cross",
		StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3},
		InitialBalance: "1000",
		FeeBps:         "1",
		SlippageBps:    "1",
		TriggerMode:    "closed_candle",
	})
	if err != nil {
		t.Fatal(err)
	}
	trading, err := store.CreateTradingTask(ctx, data.CreateTradingTask{
		Name:           integrationID("Overview Trading"),
		Type:           "paper",
		Exchange:       "binance",
		AccountID:      "paper",
		Symbol:         integrationSymbol("OT"),
		Interval:       "5m",
		StrategyID:     "ema-cross",
		StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3},
		IntentPolicy:   map[string]any{"orderIntent": "execute"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM positions WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM executions WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM orders WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM backtest_orders WHERE backtest_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM notifications WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM strategy_intents WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM trading_tasks WHERE id = $1`, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM backtest_tasks WHERE id = $1`, backtest.ID)
	})

	backtestIntentID := integrationID("si")
	tradingIntentID := integrationID("si")
	if err := store.SaveBacktestResult(ctx, data.BacktestResult{
		TaskID: backtest.ID,
		Intents: []data.StrategyIntent{
			overviewIntegrationIntent(backtestIntentID, backtest.ID, "backtest", now.Add(1*time.Hour)),
		},
		Orders: []data.BacktestOrder{
			{
				ID:         integrationID("bo"),
				BacktestID: backtest.ID,
				IntentID:   backtestIntentID,
				Side:       "buy",
				Price:      "101",
				Quantity:   "0.5",
				Status:     "filled",
				OccurredAt: now.Add(3 * time.Hour),
			},
		},
		ResultSummary: map[string]any{"totalOrders": 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTradingRunResult(ctx, data.TradingRunResult{
		TaskID: trading.ID,
		Intents: []data.StrategyIntent{
			overviewIntegrationIntent(tradingIntentID, trading.ID, "paper", now.Add(2*time.Hour)),
		},
		Orders: []data.Order{
			{
				ID:                      integrationID("ord"),
				TaskID:                  trading.ID,
				TaskType:                "paper",
				IntentID:                tradingIntentID,
				IdempotencyKey:          trading.ID + ":overview-order",
				Exchange:                trading.Exchange,
				AccountID:               trading.AccountID,
				Symbol:                  trading.Symbol,
				Side:                    "sell",
				OrderType:               "market",
				Price:                   "102",
				Quantity:                "0.25",
				Status:                  "filled",
				ExchangeResponseSummary: map[string]any{},
				CreatedAt:               now.Add(4 * time.Hour),
				UpdatedAt:               now.Add(4 * time.Hour),
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	facts, err := store.ListOverviewRecentFacts(ctx, data.OverviewRecentFactQuery{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(facts.StrategyIntents) != 2 || facts.StrategyIntents[0].ID != tradingIntentID || facts.StrategyIntents[1].ID != backtestIntentID {
		t.Fatalf("strategy intents = %#v", facts.StrategyIntents)
	}
	if facts.StrategyIntents[0].TaskName != trading.Name || facts.StrategyIntents[0].Interval != trading.Interval {
		t.Fatalf("trading task context missing: %#v", facts.StrategyIntents[0])
	}
	if len(facts.Orders) != 2 || facts.Orders[0].TaskType != "paper" || facts.Orders[1].TaskType != "backtest" {
		t.Fatalf("orders = %#v", facts.Orders)
	}
	if facts.Orders[1].TaskName != backtest.Name || facts.Orders[1].OccurredAt.IsZero() {
		t.Fatalf("backtest order context missing: %#v", facts.Orders[1])
	}

	since := now.Add(90 * time.Minute)
	filtered, err := store.ListOverviewRecentFacts(ctx, data.OverviewRecentFactQuery{Limit: 4, Since: &since})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered.StrategyIntents) != 1 || filtered.StrategyIntents[0].ID != tradingIntentID {
		t.Fatalf("filtered strategy intents = %#v", filtered.StrategyIntents)
	}
	if len(filtered.Orders) != 2 || filtered.Orders[0].TaskType != "paper" || filtered.Orders[1].TaskType != "backtest" {
		t.Fatalf("filtered orders = %#v", filtered.Orders)
	}
}

func TestIntegrationListOverviewTrendsReturnsDailyBuckets(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	base := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	backtest, err := store.CreateBacktestTask(ctx, data.CreateBacktestTask{
		Name:           integrationID("Overview Trend Backtest"),
		Exchange:       "binance",
		Symbol:         integrationSymbol("OTB"),
		Interval:       "1m",
		StrategyID:     "ema-cross",
		StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3},
		InitialBalance: "1000",
		FeeBps:         "1",
		SlippageBps:    "1",
		TriggerMode:    "closed_candle",
	})
	if err != nil {
		t.Fatal(err)
	}
	trading, err := store.CreateTradingTask(ctx, data.CreateTradingTask{
		Name:           integrationID("Overview Trend Trading"),
		Type:           "paper",
		Exchange:       "binance",
		AccountID:      "paper",
		Symbol:         integrationSymbol("OTT"),
		Interval:       "5m",
		StrategyID:     "ema-cross",
		StrategyParams: map[string]any{"fastPeriod": 2, "slowPeriod": 3},
		IntentPolicy:   map[string]any{"orderIntent": "execute", "notificationChannel": "default"},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM notification_outbox WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM notifications WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM orders WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM backtest_orders WHERE backtest_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM strategy_intents WHERE task_id IN ($1, $2)`, backtest.ID, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM trading_tasks WHERE id = $1`, trading.ID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM backtest_tasks WHERE id = $1`, backtest.ID)
	})

	intentID := integrationID("si_trend")
	if err := store.SaveBacktestResult(ctx, data.BacktestResult{
		TaskID: backtest.ID,
		Intents: []data.StrategyIntent{
			overviewIntegrationIntent(intentID, backtest.ID, "backtest", base.Add(2*time.Hour)),
		},
		Orders: []data.BacktestOrder{
			{ID: integrationID("bo_trend"), BacktestID: backtest.ID, IntentID: intentID, Side: "buy", Price: "101", Quantity: "0.5", Status: "filled", OccurredAt: base.AddDate(0, 0, 1).Add(3 * time.Hour)},
		},
		ResultSummary: map[string]any{"totalOrders": 1},
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveTradingRunResult(ctx, data.TradingRunResult{
		TaskID: trading.ID,
		Notifications: []data.Notification{
			{ID: integrationID("nt_trend"), TaskID: trading.ID, Channel: "default", Provider: "local", Target: "ops", Title: "trend failed", Body: "failed", Status: "failed", CreatedAt: base.AddDate(0, 0, 2).Add(4 * time.Hour)},
		},
	}); err != nil {
		t.Fatal(err)
	}

	trends, err := store.ListOverviewTrends(ctx, data.OverviewTrendQuery{Days: 4, From: base, To: base.AddDate(0, 0, 4)})
	if err != nil {
		t.Fatal(err)
	}
	if len(trends.Buckets) != 4 {
		t.Fatalf("bucket count = %d", len(trends.Buckets))
	}
	if trends.Buckets[0].StrategyIntents != 1 {
		t.Fatalf("day 0 bucket = %#v", trends.Buckets[0])
	}
	if trends.Buckets[1].Orders != 1 {
		t.Fatalf("day 1 bucket = %#v", trends.Buckets[1])
	}
	if trends.Buckets[2].Notifications != 1 || trends.Buckets[2].Failures != 1 {
		t.Fatalf("day 2 bucket = %#v", trends.Buckets[2])
	}
	if trends.Buckets[3].StrategyIntents != 0 || trends.Buckets[3].Orders != 0 || trends.Buckets[3].Notifications != 0 || trends.Buckets[3].Failures != 0 {
		t.Fatalf("empty bucket = %#v", trends.Buckets[3])
	}
}

func overviewIntegrationIntent(id string, taskID string, taskType string, createdAt time.Time) data.StrategyIntent {
	return data.StrategyIntent{
		ID:             id,
		TaskID:         taskID,
		TaskType:       taskType,
		StrategyID:     "ema-cross",
		IntentType:     "order",
		IdempotencyKey: taskID + ":" + id,
		Payload:        map[string]any{"side": "buy"},
		Policy:         "execute",
		Status:         "accepted",
		CreatedAt:      createdAt,
	}
}
