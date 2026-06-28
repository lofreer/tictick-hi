package api

import (
	"context"
	"sort"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListNativeCandles(
	_ context.Context,
	query data.CandleQuery,
) ([]data.Candle, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange != query.Exchange || candle.Symbol != query.Symbol || candle.Interval != query.Interval {
			continue
		}
		if query.From != nil && candle.OpenTime.Before(*query.From) {
			continue
		}
		if query.To != nil && candle.OpenTime.After(*query.To) {
			continue
		}
		matches = append(matches, candle)
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})
	if limit := data.NormalizeCandleLimit(query.Limit); len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}
