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

func (repository *fakeRepository) ListOverviewTrends(_ context.Context, query data.OverviewTrendQuery) (data.OverviewTrends, error) {
	if query.Days <= 0 {
		query.Days = data.DefaultOverviewTrendDays
	}
	buckets := make([]data.OverviewTrendBucket, 0, query.Days)
	bucketIndexes := make(map[time.Time]int, query.Days)
	for bucketStart := query.From.UTC(); bucketStart.Before(query.To); bucketStart = bucketStart.AddDate(0, 0, 1) {
		bucketIndexes[bucketStart] = len(buckets)
		buckets = append(buckets, data.OverviewTrendBucket{BucketStart: bucketStart})
	}
	add := func(at time.Time, update func(*data.OverviewTrendBucket)) {
		bucketStart := at.UTC().Truncate(24 * time.Hour)
		index, ok := bucketIndexes[bucketStart]
		if !ok {
			return
		}
		update(&buckets[index])
	}
	for _, intents := range repository.backtestIntents {
		for _, intent := range intents {
			add(intent.CreatedAt, func(bucket *data.OverviewTrendBucket) { bucket.StrategyIntents++ })
		}
	}
	for _, intents := range repository.tradingIntents {
		for _, intent := range intents {
			add(intent.CreatedAt, func(bucket *data.OverviewTrendBucket) { bucket.StrategyIntents++ })
		}
	}
	for _, orders := range repository.backtestOrders {
		for _, order := range orders {
			add(order.OccurredAt, func(bucket *data.OverviewTrendBucket) { bucket.Orders++ })
		}
	}
	for _, orders := range repository.tradingOrders {
		for _, order := range orders {
			add(order.CreatedAt, func(bucket *data.OverviewTrendBucket) { bucket.Orders++ })
		}
	}
	for _, notification := range repository.notifications {
		add(notification.CreatedAt, func(bucket *data.OverviewTrendBucket) {
			bucket.Notifications++
			if notification.Status == "failed" {
				bucket.Failures++
			}
		})
	}
	for _, task := range repository.tasks {
		if task.Status == data.TaskStatusFailed {
			add(task.UpdatedAt, func(bucket *data.OverviewTrendBucket) { bucket.Failures++ })
		}
	}
	for _, task := range repository.backtests {
		if task.Status == data.TaskStatusFailed {
			add(task.UpdatedAt, func(bucket *data.OverviewTrendBucket) { bucket.Failures++ })
		}
	}
	for _, task := range repository.tradingTasks {
		if task.Status == data.TaskStatusFailed {
			add(task.UpdatedAt, func(bucket *data.OverviewTrendBucket) { bucket.Failures++ })
		}
	}
	return data.OverviewTrends{Days: query.Days, From: query.From.UTC(), To: query.To.UTC(), Buckets: buckets}, nil
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
