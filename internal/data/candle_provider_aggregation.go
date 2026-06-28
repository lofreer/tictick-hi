package data

import (
	"context"
	"time"
)

const (
	maxAggregationBasePages   = 12
	maxAggregationBaseCandles = MaxCandleLimit * maxAggregationBasePages
)

func newAggregationBaseQuery(query CandleQuery) (CandleQuery, aggregationBaseWindow, error) {
	targetDuration, err := IntervalDuration(query.Interval)
	if err != nil {
		return CandleQuery{}, aggregationBaseWindow{}, err
	}
	baseDuration, err := IntervalDuration(baseCandleInterval)
	if err != nil {
		return CandleQuery{}, aggregationBaseWindow{}, err
	}

	baseQuery := query
	baseQuery.Interval = baseCandleInterval
	if query.From != nil {
		from := firstExpectedOpen(*query.From, targetDuration)
		baseQuery.From = &from
	}
	if query.To != nil {
		targetTo := alignTime(*query.To, targetDuration)
		to := targetTo.Add(targetDuration - baseDuration)
		baseQuery.To = &to
	}

	baseWindow := baseWindowForAggregation(query.Interval, query.Limit)
	baseQuery.Limit = baseWindow.Limit
	return baseQuery, baseWindow, nil
}

func baseWindowForAggregation(interval string, limit int) aggregationBaseWindow {
	duration, err := IntervalDuration(interval)
	if err != nil {
		return aggregationBaseWindow{Required: limit, Limit: limit}
	}
	ratio := int(duration / time.Minute)
	if ratio < 1 {
		ratio = 1
	}
	requestedLimit := NormalizeCandleLimit(limit)
	required := requestedLimit * ratio
	if required > maxAggregationBaseCandles {
		return aggregationBaseWindow{Required: required, Limit: maxAggregationBaseCandles, Limited: true}
	}
	return aggregationBaseWindow{Required: required, Limit: required}
}

func (provider *CandleProvider) listAggregationBaseCandles(
	ctx context.Context,
	query CandleQuery,
	window aggregationBaseWindow,
) ([]Candle, error) {
	if window.Limit <= 0 {
		return nil, nil
	}
	if query.From == nil {
		return provider.listLatestAggregationBaseCandles(ctx, query, window.Limit)
	}
	return provider.listForwardAggregationBaseCandles(ctx, query, window.Limit)
}

func (provider *CandleProvider) listForwardAggregationBaseCandles(
	ctx context.Context,
	query CandleQuery,
	limit int,
) ([]Candle, error) {
	from := *query.From
	candles := make([]Candle, 0, min(limit, MaxCandleLimit))
	for len(candles) < limit {
		pageLimit := aggregationPageLimit(limit - len(candles))
		pageQuery := query
		pageQuery.From = &from
		pageQuery.Limit = pageLimit

		page, err := provider.store.ListNativeCandles(ctx, pageQuery)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		candles = append(candles, page...)
		if len(page) < pageLimit {
			break
		}

		nextFrom := page[len(page)-1].OpenTime.UTC().Add(time.Minute)
		if !nextFrom.After(from) || query.To != nil && nextFrom.After(query.To.UTC()) {
			break
		}
		from = nextFrom
	}
	return candles, nil
}

func (provider *CandleProvider) listLatestAggregationBaseCandles(
	ctx context.Context,
	query CandleQuery,
	limit int,
) ([]Candle, error) {
	var to *time.Time
	if query.To != nil {
		value := query.To.UTC()
		to = &value
	}

	candles := make([]Candle, 0, min(limit, MaxCandleLimit))
	for len(candles) < limit {
		pageLimit := aggregationPageLimit(limit - len(candles))
		pageQuery := query
		pageQuery.To = to
		pageQuery.Limit = pageLimit

		page, err := provider.store.ListLatestNativeCandles(ctx, pageQuery)
		if err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		candles = append(candles, page...)
		if len(page) < pageLimit {
			break
		}

		previousTo := page[0].OpenTime.UTC().Add(-time.Minute)
		to = &previousTo
	}
	return sortedCandles(candles), nil
}

func filterCandlesInRange(candles []Candle, from *time.Time, to *time.Time) []Candle {
	if from == nil && to == nil {
		return candles
	}
	filtered := candles[:0]
	for _, candle := range candles {
		if from != nil && candle.OpenTime.Before(from.UTC()) {
			continue
		}
		if to != nil && candle.OpenTime.After(to.UTC()) {
			continue
		}
		filtered = append(filtered, candle)
	}
	return filtered
}

func trimAggregatedCandles(candles []Candle, query CandleQuery) []Candle {
	limit := NormalizeCandleLimit(query.Limit)
	if len(candles) <= limit {
		return candles
	}
	if query.From == nil {
		return candles[len(candles)-limit:]
	}
	return candles[:limit]
}

func aggregationPageLimit(remaining int) int {
	if remaining > MaxCandleLimit {
		return MaxCandleLimit
	}
	return remaining
}
