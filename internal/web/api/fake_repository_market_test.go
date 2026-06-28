package api

import (
	"context"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

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
