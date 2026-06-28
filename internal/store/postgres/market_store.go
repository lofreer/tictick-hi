package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	defaultMarketInstrumentLimit = 20
	maxMarketInstrumentLimit     = 50
)

func (store *Store) GetActiveMarketInstrument(
	ctx context.Context,
	exchange string,
	symbol string,
) (data.MarketInstrument, error) {
	var instrument data.MarketInstrument
	err := store.pool.QueryRow(ctx, `
		SELECT exchange, symbol, base_asset, quote_asset, instrument_type, status,
		       search_priority, synced_at, created_at, updated_at
		  FROM market_instruments
		 WHERE exchange = $1
		   AND symbol = $2
		   AND status = 'active'`,
		exchange,
		strings.ToUpper(strings.TrimSpace(symbol)),
	).Scan(
		&instrument.Exchange,
		&instrument.Symbol,
		&instrument.BaseAsset,
		&instrument.QuoteAsset,
		&instrument.InstrumentType,
		&instrument.Status,
		&instrument.SearchPriority,
		&instrument.SyncedAt,
		&instrument.CreatedAt,
		&instrument.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return data.MarketInstrument{}, data.ErrNotFound
		}
		return data.MarketInstrument{}, fmt.Errorf("get active market instrument: %w", err)
	}
	return instrument, nil
}

func (store *Store) ListMarketInstruments(
	ctx context.Context,
	query data.MarketInstrumentQuery,
) ([]data.MarketInstrument, error) {
	limit := normalizeMarketInstrumentLimit(query.Limit)
	search := strings.ToUpper(strings.TrimSpace(query.Query))
	prefix := search + "%"
	contains := "%" + search + "%"

	rows, err := store.pool.Query(ctx, `
		SELECT exchange, symbol, base_asset, quote_asset, instrument_type, status,
		       search_priority, synced_at, created_at, updated_at
		  FROM market_instruments
		 WHERE exchange = $1
		   AND status = 'active'
		   AND (
		     $2 = ''
		     OR symbol LIKE $3
		     OR base_asset LIKE $3
		     OR quote_asset LIKE $3
		     OR symbol LIKE $4
		   )
		 ORDER BY
		   CASE
		     WHEN $2 = '' THEN 0
		     WHEN symbol = $2 THEN 0
		     WHEN symbol LIKE $3 THEN 1
		     WHEN base_asset LIKE $3 THEN 2
		     WHEN quote_asset LIKE $3 THEN 3
		     ELSE 4
		   END,
		   search_priority,
		   symbol
		 LIMIT $5`,
		query.Exchange,
		search,
		prefix,
		contains,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list market instruments: %w", err)
	}
	defer rows.Close()

	var instruments []data.MarketInstrument
	for rows.Next() {
		var instrument data.MarketInstrument
		if err := rows.Scan(
			&instrument.Exchange,
			&instrument.Symbol,
			&instrument.BaseAsset,
			&instrument.QuoteAsset,
			&instrument.InstrumentType,
			&instrument.Status,
			&instrument.SearchPriority,
			&instrument.SyncedAt,
			&instrument.CreatedAt,
			&instrument.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan market instrument: %w", err)
		}
		instruments = append(instruments, instrument)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("collect market instruments: %w", err)
	}
	return instruments, nil
}

func (store *Store) ReplaceMarketInstruments(
	ctx context.Context,
	exchangeID string,
	instruments []data.MarketInstrument,
	syncedAt time.Time,
) (data.MarketInstrumentSyncResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.MarketInstrumentSyncResult{}, fmt.Errorf("begin market instrument sync: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	seen := make(map[string]struct{}, len(instruments))
	activeCount := 0
	for _, instrument := range instruments {
		symbol := strings.ToUpper(strings.TrimSpace(instrument.Symbol))
		baseAsset := strings.ToUpper(strings.TrimSpace(instrument.BaseAsset))
		quoteAsset := strings.ToUpper(strings.TrimSpace(instrument.QuoteAsset))
		if symbol == "" || baseAsset == "" || quoteAsset == "" {
			continue
		}
		seen[symbol] = struct{}{}
		status := normalizedInstrumentStatus(instrument.Status)
		if status == "active" {
			activeCount++
		}
		priority := instrument.SearchPriority
		if priority <= 0 {
			priority = 100
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO market_instruments (
				exchange, symbol, base_asset, quote_asset, instrument_type, status, search_priority, synced_at
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (exchange, symbol) DO UPDATE
			   SET base_asset = EXCLUDED.base_asset,
			       quote_asset = EXCLUDED.quote_asset,
			       instrument_type = EXCLUDED.instrument_type,
			       status = EXCLUDED.status,
			       search_priority = LEAST(market_instruments.search_priority, EXCLUDED.search_priority),
			       synced_at = EXCLUDED.synced_at,
			       updated_at = now()`,
			exchangeID,
			symbol,
			baseAsset,
			quoteAsset,
			normalizedInstrumentType(instrument.InstrumentType),
			status,
			priority,
			syncedAt,
		); err != nil {
			return data.MarketInstrumentSyncResult{}, fmt.Errorf("upsert market instrument: %w", err)
		}
	}

	symbols := make([]string, 0, len(seen))
	for symbol := range seen {
		symbols = append(symbols, symbol)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE market_instruments
		   SET status = 'inactive',
		       synced_at = $2,
		       updated_at = now()
		 WHERE exchange = $1
		   AND status = 'active'
		   AND NOT (symbol = ANY($3))`,
		exchangeID,
		syncedAt,
		symbols,
	)
	if err != nil {
		return data.MarketInstrumentSyncResult{}, fmt.Errorf("deactivate stale market instruments: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.MarketInstrumentSyncResult{}, fmt.Errorf("commit market instrument sync: %w", err)
	}
	return data.MarketInstrumentSyncResult{
		Exchange:      exchangeID,
		ActiveCount:   activeCount,
		InactiveCount: int(tag.RowsAffected()),
		SyncedAt:      syncedAt,
	}, nil
}

func normalizeMarketInstrumentLimit(limit int) int {
	if limit <= 0 {
		return defaultMarketInstrumentLimit
	}
	if limit > maxMarketInstrumentLimit {
		return maxMarketInstrumentLimit
	}
	return limit
}

func normalizedInstrumentStatus(status string) string {
	if strings.EqualFold(strings.TrimSpace(status), "inactive") {
		return "inactive"
	}
	return "active"
}

func normalizedInstrumentType(instrumentType string) string {
	instrumentType = strings.ToLower(strings.TrimSpace(instrumentType))
	if instrumentType == "" {
		return "spot"
	}
	return instrumentType
}
