package api

import (
	"context"
	"slices"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func defaultFakeMarketInstruments(now time.Time) []data.MarketInstrument {
	return []data.MarketInstrument{{
		Exchange:       "binance",
		Symbol:         "BTCUSDT",
		BaseAsset:      "BTC",
		QuoteAsset:     "USDT",
		InstrumentType: "spot",
		Status:         "active",
		SearchPriority: 1,
		CreatedAt:      now,
		UpdatedAt:      now,
	}}
}

func (repository *fakeRepository) GetActiveMarketInstrument(
	_ context.Context,
	exchange string,
	symbol string,
) (data.MarketInstrument, error) {
	index := slices.IndexFunc(repository.marketInstruments, func(instrument data.MarketInstrument) bool {
		return instrument.Exchange == exchange && instrument.Symbol == symbol && instrument.Status == "active"
	})
	if index < 0 {
		return data.MarketInstrument{}, data.ErrNotFound
	}
	return repository.marketInstruments[index], nil
}

func (repository *fakeRepository) ListMarketInstruments(
	_ context.Context,
	query data.MarketInstrumentQuery,
) ([]data.MarketInstrument, error) {
	search := strings.ToUpper(strings.TrimSpace(query.Query))
	limit := query.Limit
	if limit <= 0 || limit > 50 {
		limit = 50
	}
	var results []data.MarketInstrument
	for _, instrument := range repository.marketInstruments {
		if instrument.Exchange != query.Exchange || instrument.Status != "active" {
			continue
		}
		if search != "" &&
			!strings.Contains(instrument.Symbol, search) &&
			!strings.HasPrefix(instrument.BaseAsset, search) &&
			!strings.HasPrefix(instrument.QuoteAsset, search) {
			continue
		}
		results = append(results, instrument)
		if len(results) >= limit {
			return results, nil
		}
	}
	return results, nil
}

func (repository *fakeRepository) ReplaceMarketInstruments(
	_ context.Context,
	exchangeID string,
	instruments []data.MarketInstrument,
	syncedAt time.Time,
) (data.MarketInstrumentSyncResult, error) {
	active := 0
	incoming := map[string]data.MarketInstrument{}
	for _, instrument := range instruments {
		instrument.Exchange = exchangeID
		instrument.SyncedAt = &syncedAt
		incoming[instrument.Symbol] = instrument
		if instrument.Status == "active" {
			active++
		}
	}
	inactive := 0
	for index := range repository.marketInstruments {
		instrument := &repository.marketInstruments[index]
		if instrument.Exchange != exchangeID {
			continue
		}
		if replacement, ok := incoming[instrument.Symbol]; ok {
			repository.marketInstruments[index] = replacement
			delete(incoming, instrument.Symbol)
			continue
		}
		if instrument.Status == "active" {
			instrument.Status = "inactive"
			instrument.SyncedAt = &syncedAt
			inactive++
		}
	}
	for _, instrument := range incoming {
		repository.marketInstruments = append(repository.marketInstruments, instrument)
	}
	return data.MarketInstrumentSyncResult{
		Exchange:      exchangeID,
		ActiveCount:   active,
		InactiveCount: inactive,
		SyncedAt:      syncedAt,
	}, nil
}
