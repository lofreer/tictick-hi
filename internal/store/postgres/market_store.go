package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	defaultMarketInstrumentLimit = 20
	maxMarketInstrumentLimit     = 50
)

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

func normalizeMarketInstrumentLimit(limit int) int {
	if limit <= 0 {
		return defaultMarketInstrumentLimit
	}
	if limit > maxMarketInstrumentLimit {
		return maxMarketInstrumentLimit
	}
	return limit
}
