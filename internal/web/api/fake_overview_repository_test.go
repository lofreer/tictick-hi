package api

import (
	"context"
	"sort"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListOverviewRecentFacts(_ context.Context, query data.OverviewRecentFactQuery) (data.OverviewRecentFacts, error) {
	if query.Limit <= 0 {
		query.Limit = data.DefaultOverviewRecentFactLimit
	}
	intents := make([]data.OverviewStrategyIntentFact, 0)
	for _, task := range repository.backtests {
		for _, intent := range repository.backtestIntents[task.ID] {
			if !overviewFactAtOrAfter(intent.CreatedAt, query.Since) {
				continue
			}
			intents = append(intents, overviewIntentFact(task.Name, task.Exchange, task.Symbol, task.Interval, intent))
		}
	}
	for _, task := range repository.tradingTasks {
		for _, intent := range repository.tradingIntents[task.ID] {
			if !overviewFactAtOrAfter(intent.CreatedAt, query.Since) {
				continue
			}
			intents = append(intents, overviewIntentFact(task.Name, task.Exchange, task.Symbol, task.Interval, intent))
		}
	}
	sort.Slice(intents, func(left, right int) bool {
		return intents[left].CreatedAt.After(intents[right].CreatedAt)
	})
	if len(intents) > query.Limit {
		intents = intents[:query.Limit]
	}

	orders := make([]data.OverviewOrderFact, 0)
	for _, task := range repository.backtests {
		for _, order := range repository.backtestOrders[task.ID] {
			if !overviewFactAtOrAfter(order.OccurredAt, query.Since) {
				continue
			}
			orders = append(orders, data.OverviewOrderFact{
				ID:         order.ID,
				TaskID:     task.ID,
				TaskType:   "backtest",
				TaskName:   task.Name,
				Exchange:   task.Exchange,
				Symbol:     task.Symbol,
				Interval:   task.Interval,
				IntentID:   order.IntentID,
				Side:       order.Side,
				Price:      order.Price,
				Quantity:   order.Quantity,
				Status:     order.Status,
				OccurredAt: order.OccurredAt,
			})
		}
	}
	for _, task := range repository.tradingTasks {
		for _, order := range repository.tradingOrders[task.ID] {
			if !overviewFactAtOrAfter(order.CreatedAt, query.Since) {
				continue
			}
			orders = append(orders, data.OverviewOrderFact{
				ID:         order.ID,
				TaskID:     task.ID,
				TaskType:   task.Type,
				TaskName:   task.Name,
				Exchange:   task.Exchange,
				Symbol:     task.Symbol,
				Interval:   task.Interval,
				IntentID:   order.IntentID,
				Side:       order.Side,
				Price:      order.Price,
				Quantity:   order.Quantity,
				Status:     order.Status,
				OccurredAt: order.CreatedAt,
			})
		}
	}
	sort.Slice(orders, func(left, right int) bool {
		return orders[left].OccurredAt.After(orders[right].OccurredAt)
	})
	if len(orders) > query.Limit {
		orders = orders[:query.Limit]
	}

	return data.OverviewRecentFacts{StrategyIntents: intents, Orders: orders}, nil
}

func overviewFactAtOrAfter(at time.Time, since *time.Time) bool {
	return since == nil || !at.Before(*since)
}

func overviewIntentFact(taskName string, exchange string, symbol string, interval string, intent data.StrategyIntent) data.OverviewStrategyIntentFact {
	return data.OverviewStrategyIntentFact{
		ID:         intent.ID,
		TaskID:     intent.TaskID,
		TaskType:   intent.TaskType,
		TaskName:   taskName,
		Exchange:   exchange,
		Symbol:     symbol,
		Interval:   interval,
		StrategyID: intent.StrategyID,
		IntentType: intent.IntentType,
		Policy:     intent.Policy,
		Status:     intent.Status,
		CreatedAt:  intent.CreatedAt,
	}
}
