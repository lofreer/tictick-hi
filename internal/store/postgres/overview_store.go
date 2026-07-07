package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ListOverviewRecentFacts(ctx context.Context, query data.OverviewRecentFactQuery) (data.OverviewRecentFacts, error) {
	if query.Limit <= 0 {
		query.Limit = data.DefaultOverviewRecentFactLimit
	}
	intents, err := store.listOverviewStrategyIntentFacts(ctx, query)
	if err != nil {
		return data.OverviewRecentFacts{}, err
	}
	orders, err := store.listOverviewOrderFacts(ctx, query)
	if err != nil {
		return data.OverviewRecentFacts{}, err
	}
	return data.OverviewRecentFacts{
		StrategyIntents: intents,
		Orders:          orders,
	}, nil
}

func (store *Store) ListOverviewTrends(ctx context.Context, query data.OverviewTrendQuery) (data.OverviewTrends, error) {
	if query.Days <= 0 {
		query.Days = data.DefaultOverviewTrendDays
	}
	if query.From.IsZero() || query.To.IsZero() {
		to := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		query.To = to
		query.From = to.AddDate(0, 0, -query.Days)
	}
	rows, err := store.pool.Query(ctx, `
		WITH buckets AS (
			SELECT generate_series($1::timestamptz, $2::timestamptz - interval '1 day', interval '1 day') AS bucket_start
		),
		intent_counts AS (
			SELECT date_trunc('day', created_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket_start, count(*) AS count
			  FROM strategy_intents
			 WHERE created_at >= $1
			   AND created_at < $2
			 GROUP BY 1
		),
		order_counts AS (
			SELECT date_trunc('day', occurred_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket_start, count(*) AS count
			  FROM (
				SELECT occurred_at
				  FROM backtest_orders
				UNION ALL
				SELECT created_at AS occurred_at
				  FROM orders
			  ) facts
			 WHERE occurred_at >= $1
			   AND occurred_at < $2
			 GROUP BY 1
		),
		notification_counts AS (
			SELECT date_trunc('day', created_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket_start, count(*) AS count
			  FROM notifications
			 WHERE created_at >= $1
			   AND created_at < $2
			 GROUP BY 1
		),
		failure_counts AS (
			SELECT date_trunc('day', failed_at AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS bucket_start, count(*) AS count
			  FROM (
				SELECT updated_at AS failed_at FROM data_sync_tasks WHERE status = 'failed'
				UNION ALL
				SELECT updated_at AS failed_at FROM backtest_tasks WHERE status = 'failed'
				UNION ALL
				SELECT updated_at AS failed_at FROM trading_tasks WHERE status = 'failed'
				UNION ALL
				SELECT created_at AS failed_at FROM notifications WHERE status = 'failed'
			  ) failures
			 WHERE failed_at >= $1
			   AND failed_at < $2
			 GROUP BY 1
		)
		SELECT buckets.bucket_start,
		       COALESCE(intent_counts.count, 0),
		       COALESCE(order_counts.count, 0),
		       COALESCE(notification_counts.count, 0),
		       COALESCE(failure_counts.count, 0)
		  FROM buckets
		  LEFT JOIN intent_counts USING (bucket_start)
		  LEFT JOIN order_counts USING (bucket_start)
		  LEFT JOIN notification_counts USING (bucket_start)
		  LEFT JOIN failure_counts USING (bucket_start)
		 ORDER BY buckets.bucket_start ASC`, query.From.UTC(), query.To.UTC())
	if err != nil {
		return data.OverviewTrends{}, fmt.Errorf("list overview trends: %w", err)
	}
	defer rows.Close()

	buckets, err := pgx.CollectRows(rows, scanOverviewTrendBucket)
	if err != nil {
		return data.OverviewTrends{}, fmt.Errorf("scan overview trends: %w", err)
	}
	return data.OverviewTrends{
		Days:    query.Days,
		From:    query.From.UTC(),
		To:      query.To.UTC(),
		Buckets: buckets,
	}, nil
}

func (store *Store) listOverviewStrategyIntentFacts(ctx context.Context, query data.OverviewRecentFactQuery) ([]data.OverviewStrategyIntentFact, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT si.id, si.task_id, si.task_type, COALESCE(bt.name, tt.name) AS task_name,
		       COALESCE(bt.exchange, tt.exchange) AS exchange,
		       COALESCE(bt.symbol, tt.symbol) AS symbol,
		       COALESCE(bt.interval, tt.interval) AS interval,
		       si.strategy_id, si.intent_type, si.policy, si.status, si.created_at
		  FROM strategy_intents si
		  LEFT JOIN backtest_tasks bt
		    ON si.task_type = 'backtest'
		   AND si.task_id = bt.id
		  LEFT JOIN trading_tasks tt
		    ON si.task_type IN ('paper', 'live')
		   AND si.task_id = tt.id
		 WHERE ($2::timestamptz IS NULL OR si.created_at >= $2)
		   AND ((si.task_type = 'backtest' AND bt.id IS NOT NULL)
		    OR (si.task_type IN ('paper', 'live') AND tt.id IS NOT NULL))
		 ORDER BY si.created_at DESC
		 LIMIT $1`, query.Limit, overviewFactSinceParam(query.Since))
	if err != nil {
		return nil, fmt.Errorf("list overview strategy intent facts: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanOverviewStrategyIntentFact)
}

func (store *Store) listOverviewOrderFacts(ctx context.Context, query data.OverviewRecentFactQuery) ([]data.OverviewOrderFact, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, task_type, task_name, exchange, symbol, interval,
		       COALESCE(intent_id, ''), side, price, quantity, status, occurred_at
		  FROM (
			SELECT bo.id, bo.backtest_id AS task_id, 'backtest' AS task_type,
			       bt.name AS task_name, bt.exchange, bt.symbol, bt.interval,
			       bo.intent_id, bo.side, bo.price::text AS price, bo.quantity::text AS quantity,
			       bo.status, bo.occurred_at
			  FROM backtest_orders bo
			  JOIN backtest_tasks bt
			    ON bo.backtest_id = bt.id
			UNION ALL
			SELECT o.id, o.task_id, o.task_type, tt.name AS task_name,
			       tt.exchange, tt.symbol, tt.interval, o.intent_id,
			       o.side, o.price::text AS price, o.quantity::text AS quantity,
			       o.status, o.created_at AS occurred_at
			  FROM orders o
			  JOIN trading_tasks tt
			    ON o.task_id = tt.id
		  ) facts
		 WHERE ($2::timestamptz IS NULL OR occurred_at >= $2)
		 ORDER BY occurred_at DESC
		 LIMIT $1`, query.Limit, overviewFactSinceParam(query.Since))
	if err != nil {
		return nil, fmt.Errorf("list overview order facts: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanOverviewOrderFact)
}

func overviewFactSinceParam(since *time.Time) any {
	if since == nil {
		return nil
	}
	return since.UTC()
}

func scanOverviewStrategyIntentFact(row pgx.CollectableRow) (data.OverviewStrategyIntentFact, error) {
	var fact data.OverviewStrategyIntentFact
	err := row.Scan(
		&fact.ID,
		&fact.TaskID,
		&fact.TaskType,
		&fact.TaskName,
		&fact.Exchange,
		&fact.Symbol,
		&fact.Interval,
		&fact.StrategyID,
		&fact.IntentType,
		&fact.Policy,
		&fact.Status,
		&fact.CreatedAt,
	)
	return fact, err
}

func scanOverviewOrderFact(row pgx.CollectableRow) (data.OverviewOrderFact, error) {
	var fact data.OverviewOrderFact
	err := row.Scan(
		&fact.ID,
		&fact.TaskID,
		&fact.TaskType,
		&fact.TaskName,
		&fact.Exchange,
		&fact.Symbol,
		&fact.Interval,
		&fact.IntentID,
		&fact.Side,
		&fact.Price,
		&fact.Quantity,
		&fact.Status,
		&fact.OccurredAt,
	)
	return fact, err
}

func scanOverviewTrendBucket(row pgx.CollectableRow) (data.OverviewTrendBucket, error) {
	var bucket data.OverviewTrendBucket
	err := row.Scan(
		&bucket.BucketStart,
		&bucket.StrategyIntents,
		&bucket.Orders,
		&bucket.Notifications,
		&bucket.Failures,
	)
	return bucket, err
}
