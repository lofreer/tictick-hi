package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ListOverviewRecentFacts(ctx context.Context, limit int) (data.OverviewRecentFacts, error) {
	if limit <= 0 {
		limit = data.DefaultOverviewRecentFactLimit
	}
	intents, err := store.listOverviewStrategyIntentFacts(ctx, limit)
	if err != nil {
		return data.OverviewRecentFacts{}, err
	}
	orders, err := store.listOverviewOrderFacts(ctx, limit)
	if err != nil {
		return data.OverviewRecentFacts{}, err
	}
	return data.OverviewRecentFacts{
		StrategyIntents: intents,
		Orders:          orders,
	}, nil
}

func (store *Store) listOverviewStrategyIntentFacts(ctx context.Context, limit int) ([]data.OverviewStrategyIntentFact, error) {
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
		 WHERE (si.task_type = 'backtest' AND bt.id IS NOT NULL)
		    OR (si.task_type IN ('paper', 'live') AND tt.id IS NOT NULL)
		 ORDER BY si.created_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list overview strategy intent facts: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanOverviewStrategyIntentFact)
}

func (store *Store) listOverviewOrderFacts(ctx context.Context, limit int) ([]data.OverviewOrderFact, error) {
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
		 ORDER BY occurred_at DESC
		 LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list overview order facts: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanOverviewOrderFact)
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
