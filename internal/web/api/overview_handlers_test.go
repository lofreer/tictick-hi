package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestOverviewRecentFactsRouteReturnsGlobalFacts(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	oldTaskTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	newTaskTime := oldTaskTime.Add(1 * time.Hour)
	repository.backtests = []data.BacktestTask{
		{
			ID:            "bt_old",
			Name:          "Old backtest",
			Exchange:      "binance",
			Symbol:        "BTCUSDT",
			Interval:      "1m",
			StrategyID:    "ema-cross",
			Status:        data.TaskStatusSucceeded,
			ResultSummary: map[string]any{},
			CreatedAt:     oldTaskTime,
			UpdatedAt:     oldTaskTime,
		},
		{
			ID:            "bt_new",
			Name:          "New backtest",
			Exchange:      "binance",
			Symbol:        "ETHUSDT",
			Interval:      "5m",
			StrategyID:    "ema-cross",
			Status:        data.TaskStatusSucceeded,
			ResultSummary: map[string]any{},
			CreatedAt:     newTaskTime,
			UpdatedAt:     newTaskTime,
		},
	}
	repository.tradingTasks = []data.TradingTask{
		{
			ID:        "tt_paper",
			Name:      "Paper trend",
			Type:      "paper",
			Exchange:  "binance",
			AccountID: "paper",
			Symbol:    "SOLUSDT",
			Interval:  "15m",
			Status:    data.TaskStatusRunning,
			CreatedAt: newTaskTime,
			UpdatedAt: newTaskTime,
		},
	}
	repository.backtestIntents["bt_old"] = []data.StrategyIntent{
		strategyIntent("si_old_latest", "bt_old", "backtest", "accepted", oldTaskTime.Add(3*time.Hour)),
	}
	repository.backtestIntents["bt_new"] = []data.StrategyIntent{
		strategyIntent("si_new_older", "bt_new", "backtest", "accepted", oldTaskTime.Add(2*time.Hour)),
	}
	repository.tradingIntents["tt_paper"] = []data.StrategyIntent{
		strategyIntent("si_trading", "tt_paper", "paper", "accepted", oldTaskTime.Add(4*time.Hour)),
	}
	repository.backtestOrders["bt_old"] = []data.BacktestOrder{
		backtestOrder("bo_old_latest", "bt_old", "buy", "101", oldTaskTime.Add(5*time.Hour)),
	}
	repository.tradingOrders["tt_paper"] = []data.Order{
		tradingOrder("ord_trading", "tt_paper", "paper", "sell", "99", oldTaskTime.Add(4*time.Hour)),
	}

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/overview/recent-facts?limit=2", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var facts data.OverviewRecentFacts
	if err := json.Unmarshal(recorder.Body.Bytes(), &facts); err != nil {
		t.Fatalf("decode facts: %v", err)
	}
	if len(facts.StrategyIntents) != 2 || facts.StrategyIntents[0].ID != "si_trading" || facts.StrategyIntents[1].ID != "si_old_latest" {
		t.Fatalf("strategy intents = %#v", facts.StrategyIntents)
	}
	if facts.StrategyIntents[1].TaskName != "Old backtest" || facts.StrategyIntents[1].Symbol != "BTCUSDT" {
		t.Fatalf("old task context missing from global intent fact: %#v", facts.StrategyIntents[1])
	}
	if len(facts.Orders) != 2 || facts.Orders[0].ID != "bo_old_latest" || facts.Orders[1].ID != "ord_trading" {
		t.Fatalf("orders = %#v", facts.Orders)
	}
	if facts.Orders[0].TaskType != "backtest" || facts.Orders[1].TaskType != "paper" {
		t.Fatalf("order task types = %#v", facts.Orders)
	}
}

func TestOverviewRecentFactsRouteFiltersBySince(t *testing.T) {
	repository, server, auth := newAuthenticatedTestServer(t)
	baseTime := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository.backtests = []data.BacktestTask{
		{ID: "bt_old", Name: "Old backtest", Exchange: "binance", Symbol: "BTCUSDT", Interval: "1m", Status: data.TaskStatusSucceeded, ResultSummary: map[string]any{}, CreatedAt: baseTime, UpdatedAt: baseTime},
	}
	repository.tradingTasks = []data.TradingTask{
		{ID: "tt_paper", Name: "Paper trend", Type: "paper", Exchange: "binance", AccountID: "paper", Symbol: "SOLUSDT", Interval: "15m", Status: data.TaskStatusRunning, CreatedAt: baseTime, UpdatedAt: baseTime},
	}
	repository.backtestIntents["bt_old"] = []data.StrategyIntent{
		strategyIntent("si_before", "bt_old", "backtest", "accepted", baseTime.Add(3*time.Hour)),
	}
	repository.tradingIntents["tt_paper"] = []data.StrategyIntent{
		strategyIntent("si_since", "tt_paper", "paper", "accepted", baseTime.Add(4*time.Hour)),
	}
	repository.backtestOrders["bt_old"] = []data.BacktestOrder{
		backtestOrder("bo_after", "bt_old", "buy", "101", baseTime.Add(5*time.Hour)),
	}
	repository.tradingOrders["tt_paper"] = []data.Order{
		tradingOrder("ord_since", "tt_paper", "paper", "sell", "99", baseTime.Add(4*time.Hour)),
	}

	since := url.QueryEscape(baseTime.Add(4 * time.Hour).Format(time.RFC3339))
	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/overview/recent-facts?limit=8&since="+since, "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var facts data.OverviewRecentFacts
	if err := json.Unmarshal(recorder.Body.Bytes(), &facts); err != nil {
		t.Fatalf("decode facts: %v", err)
	}
	if len(facts.StrategyIntents) != 1 || facts.StrategyIntents[0].ID != "si_since" {
		t.Fatalf("strategy intents = %#v", facts.StrategyIntents)
	}
	if len(facts.Orders) != 2 || facts.Orders[0].ID != "bo_after" || facts.Orders[1].ID != "ord_since" {
		t.Fatalf("orders = %#v", facts.Orders)
	}
}

func TestOverviewRecentFactsRouteRejectsOversizedLimit(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/overview/recent-facts?limit=51", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestOverviewRecentFactsRouteRejectsInvalidSince(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/overview/recent-facts?since=not-a-time", "")
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func strategyIntent(id string, taskID string, taskType string, status string, createdAt time.Time) data.StrategyIntent {
	return data.StrategyIntent{
		ID:             id,
		TaskID:         taskID,
		TaskType:       taskType,
		StrategyID:     "ema-cross",
		IntentType:     "order",
		IdempotencyKey: taskID + ":" + id,
		Payload:        map[string]any{"side": "buy"},
		Policy:         "execute",
		Status:         status,
		CreatedAt:      createdAt,
	}
}

func backtestOrder(id string, backtestID string, side string, price string, occurredAt time.Time) data.BacktestOrder {
	return data.BacktestOrder{
		ID:         id,
		BacktestID: backtestID,
		IntentID:   "si_" + id,
		Side:       side,
		Price:      price,
		Quantity:   "1",
		Status:     "filled",
		OccurredAt: occurredAt,
	}
}

func tradingOrder(id string, taskID string, taskType string, side string, price string, createdAt time.Time) data.Order {
	return data.Order{
		ID:                      id,
		TaskID:                  taskID,
		TaskType:                taskType,
		IntentID:                "si_" + id,
		IdempotencyKey:          taskID + ":" + id,
		Exchange:                "binance",
		AccountID:               "paper",
		Symbol:                  "SOLUSDT",
		Side:                    side,
		OrderType:               "market",
		Price:                   price,
		Quantity:                "1",
		Status:                  "filled",
		ExchangeResponseSummary: map[string]any{},
		CreatedAt:               createdAt,
		UpdatedAt:               createdAt,
	}
}
