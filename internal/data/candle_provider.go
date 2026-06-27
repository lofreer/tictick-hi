package data

import (
	"context"
	"time"
)

const baseCandleInterval = "1m"

type CandleProvider struct {
	store NativeCandleStore
}

func NewCandleProvider(store NativeCandleStore) *CandleProvider {
	return &CandleProvider{store: store}
}

func (provider *CandleProvider) GetCandles(ctx context.Context, query CandleQuery) (CandleResult, error) {
	nativeCandles, err := provider.store.ListNativeCandles(ctx, query)
	if err != nil {
		return CandleResult{}, err
	}

	nativeGaps, err := DetectCandleGaps(nativeCandles, query.Interval)
	if err != nil {
		return CandleResult{}, err
	}
	if query.Interval == baseCandleInterval || len(nativeCandles) > 0 && len(nativeGaps) == 0 {
		return candleResult(query, nativeCandles, CandleSourceNative, query.Interval, nativeGaps), nil
	}

	baseQuery := query
	baseQuery.Interval = baseCandleInterval
	baseWindow := baseWindowForAggregation(query.Interval, query.Limit)
	baseQuery.Limit = baseWindow.Limit
	baseCandles, err := provider.store.ListNativeCandles(ctx, baseQuery)
	if err != nil {
		return CandleResult{}, err
	}

	baseGaps, err := DetectCandleGaps(baseCandles, baseCandleInterval)
	if err != nil {
		return CandleResult{}, err
	}
	aggregated, err := AggregateCandles(baseCandles, query.Interval)
	if err != nil {
		return CandleResult{}, err
	}
	if query.Limit > 0 && len(aggregated) > query.Limit {
		aggregated = aggregated[:query.Limit]
	}

	result := candleResult(query, aggregated, CandleSourceAggregated, baseCandleInterval, baseGaps)
	result.Coverage.RequiredBaseCandles = baseWindow.Required
	result.Coverage.BaseLimit = baseWindow.Limit
	result.Coverage.ReturnedBaseCandles = len(baseCandles)
	result.Coverage.LimitedByBaseWindow = baseWindow.Limited
	if baseWindow.Limited && len(aggregated) < result.Coverage.RequestedLimit {
		result.Health = CandleHealthInsufficient
	}
	if len(baseCandles) == 0 {
		if len(nativeCandles) > 0 {
			return candleResult(query, nativeCandles, CandleSourceNative, query.Interval, nativeGaps), nil
		}
		result.Source = CandleSourceNone
		result.BaseInterval = ""
	}
	if len(aggregated) == 0 && len(baseGaps) > 0 {
		result.Health = CandleHealthGap
	}
	return result, nil
}

func DetectCandleGaps(candles []Candle, interval string) ([]CandleGap, error) {
	duration, err := IntervalDuration(interval)
	if err != nil {
		return nil, err
	}
	if len(candles) < 2 {
		return nil, nil
	}

	ordered := sortedCandles(candles)
	gaps := make([]CandleGap, 0)
	for index := 1; index < len(ordered); index++ {
		expected := ordered[index-1].OpenTime.UTC().Add(duration)
		actual := ordered[index].OpenTime.UTC()
		if !actual.After(expected) {
			continue
		}
		missing := int(actual.Sub(expected) / duration)
		if missing < 1 {
			missing = 1
		}
		gaps = append(gaps, CandleGap{
			From:           expected,
			To:             actual,
			MissingCandles: missing,
		})
	}
	return gaps, nil
}

func candleResult(
	query CandleQuery,
	candles []Candle,
	source CandleSource,
	baseInterval string,
	gaps []CandleGap,
) CandleResult {
	result := CandleResult{
		Candles:           candles,
		Source:            source,
		RequestedInterval: query.Interval,
		BaseInterval:      baseInterval,
		Health:            candleHealth(candles, gaps),
		Gaps:              gaps,
		Coverage: CandleCoverage{
			RequestedLimit:  NormalizeCandleLimit(query.Limit),
			ReturnedCandles: len(candles),
		},
	}
	if len(candles) == 0 {
		result.Source = CandleSourceNone
		result.BaseInterval = ""
	}
	return result
}

func candleHealth(candles []Candle, gaps []CandleGap) CandleHealth {
	if len(gaps) > 0 {
		return CandleHealthGap
	}
	if len(candles) == 0 {
		return CandleHealthInsufficient
	}
	return CandleHealthOK
}

type aggregationBaseWindow struct {
	Required int
	Limit    int
	Limited  bool
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
	if required > MaxCandleLimit {
		return aggregationBaseWindow{Required: required, Limit: MaxCandleLimit, Limited: true}
	}
	return aggregationBaseWindow{Required: required, Limit: required}
}
