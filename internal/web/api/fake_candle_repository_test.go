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

func (repository *fakeRepository) ScanMarketCandleGaps(
	_ context.Context,
	query data.MarketCandleGapScanQuery,
) (data.MarketCandleGapScan, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange == query.Exchange && candle.Symbol == query.Symbol && candle.Interval == query.Interval {
			matches = append(matches, candle)
		}
	}
	sort.Slice(matches, func(left int, right int) bool {
		return matches[left].OpenTime.Before(matches[right].OpenTime)
	})

	result := data.MarketCandleGapScan{
		Exchange: query.Exchange,
		Symbol:   query.Symbol,
		Interval: query.Interval,
		Gaps:     []data.CandleGap{},
		Window: data.CandleWindow{
			Count: len(matches),
		},
	}
	if len(matches) > 0 {
		from := matches[0].OpenTime.UTC()
		to := matches[len(matches)-1].OpenTime.UTC()
		result.Window.From = &from
		result.Window.To = &to
	}

	gaps, err := data.DetectCandleGaps(matches, query.Interval)
	if err != nil {
		return data.MarketCandleGapScan{}, err
	}
	limit := data.NormalizeMarketCandleGapScanLimit(query.Limit)
	result.TotalCount = len(gaps)
	result.Limited = len(gaps) > limit
	if len(gaps) > limit {
		gaps = gaps[:limit]
	}
	result.Gaps = gaps
	result.ReturnedCount = len(gaps)
	return result, nil
}
