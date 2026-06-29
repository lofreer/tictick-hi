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
	if err := validateCandleSeries(nativeCandles, query.Interval); err != nil {
		return invalidCandleResult(query, CandleSourceNative, query.Interval, nativeCandles, query.Interval, "invalid_native_series", err), nil
	}

	nativeGaps, err := DetectCandleGapsInRange(nativeCandles, query.Interval, query.From, query.To)
	if err != nil {
		return CandleResult{}, err
	}
	if query.Interval == baseCandleInterval || len(nativeCandles) > 0 && len(nativeGaps) == 0 {
		return provider.withPagination(ctx, query, candleResult(query, nativeCandles, CandleSourceNative, query.Interval, nativeGaps))
	}

	baseQuery, baseWindow, err := newAggregationBaseQuery(query)
	if err != nil {
		return CandleResult{}, err
	}
	baseCandles, err := provider.listAggregationBaseCandles(ctx, baseQuery, baseWindow)
	if err != nil {
		return CandleResult{}, err
	}
	if err := validateCandleSeries(baseCandles, baseCandleInterval); err != nil {
		result := invalidCandleResult(query, CandleSourceAggregated, baseCandleInterval, baseCandles, baseCandleInterval, "invalid_aggregation_base_series", err)
		result.Coverage.RequiredBaseCandles = baseWindow.Required
		result.Coverage.BaseLimit = baseWindow.Limit
		result.Coverage.ReturnedBaseCandles = len(baseCandles)
		result.Coverage.LimitedByBaseWindow = baseWindow.Limited && len(baseCandles) >= baseWindow.Limit
		return result, nil
	}

	baseGaps, err := DetectCandleGapsInRange(baseCandles, baseCandleInterval, baseQuery.From, baseQuery.To)
	if err != nil {
		return CandleResult{}, err
	}
	aggregated, err := AggregateCandles(baseCandles, query.Interval)
	if err != nil {
		return CandleResult{}, err
	}
	aggregated = filterCandlesInRange(aggregated, query.From, query.To)
	aggregated = trimAggregatedCandles(aggregated, query)

	result := candleResult(query, aggregated, CandleSourceAggregated, baseCandleInterval, baseGaps)
	result.Coverage.RequiredBaseCandles = baseWindow.Required
	result.Coverage.BaseLimit = baseWindow.Limit
	result.Coverage.ReturnedBaseCandles = len(baseCandles)
	result.Coverage.LimitedByBaseWindow = baseWindow.Limited && len(baseCandles) >= baseWindow.Limit
	if result.Coverage.LimitedByBaseWindow && len(aggregated) < result.Coverage.RequestedLimit {
		result.Health = CandleHealthInsufficient
	}
	if len(baseCandles) == 0 {
		if len(nativeCandles) > 0 {
			return provider.withPagination(ctx, query, candleResult(query, nativeCandles, CandleSourceNative, query.Interval, nativeGaps))
		}
		result.Source = CandleSourceNone
		result.BaseInterval = ""
	}
	if len(aggregated) == 0 && len(baseGaps) > 0 {
		result.Health = CandleHealthGap
	}
	return provider.withPagination(ctx, query, result)
}

func DetectCandleGaps(candles []Candle, interval string) ([]CandleGap, error) {
	return DetectCandleGapsInRange(candles, interval, nil, nil)
}

func DetectCandleGapsInRange(candles []Candle, interval string, from *time.Time, to *time.Time) ([]CandleGap, error) {
	duration, err := IntervalDuration(interval)
	if err != nil {
		return nil, err
	}
	if len(candles) == 0 {
		if from == nil || to == nil {
			return nil, nil
		}
		firstExpected := firstExpectedOpen(*from, duration)
		lastExpected := alignTime(*to, duration)
		if lastExpected.Before(firstExpected) {
			return nil, nil
		}
		return []CandleGap{newCandleGap(firstExpected, lastExpected.Add(duration), duration)}, nil
	}

	ordered := sortedCandles(candles)
	gaps := make([]CandleGap, 0)
	if from != nil {
		firstExpected := firstExpectedOpen(*from, duration)
		firstOpen := ordered[0].OpenTime.UTC()
		if firstOpen.After(firstExpected) {
			gaps = append(gaps, newCandleGap(firstExpected, firstOpen, duration))
		}
	}
	for index := 1; index < len(ordered); index++ {
		expected := ordered[index-1].OpenTime.UTC().Add(duration)
		actual := ordered[index].OpenTime.UTC()
		if !actual.After(expected) {
			continue
		}
		gaps = append(gaps, newCandleGap(expected, actual, duration))
	}
	if to != nil {
		lastOpen := ordered[len(ordered)-1].OpenTime.UTC()
		lastExpected := alignTime(*to, duration)
		nextExpected := lastOpen.Add(duration)
		if !lastExpected.Before(nextExpected) {
			gaps = append(gaps, newCandleGap(nextExpected, lastExpected.Add(duration), duration))
		}
	}
	return gaps, nil
}

func firstExpectedOpen(value time.Time, duration time.Duration) time.Time {
	aligned := alignTime(value, duration)
	if aligned.Before(value.UTC()) {
		return aligned.Add(duration)
	}
	return aligned
}

func newCandleGap(from time.Time, to time.Time, duration time.Duration) CandleGap {
	missing := int(to.Sub(from) / duration)
	if missing < 1 {
		missing = 1
	}
	return CandleGap{
		From:           from,
		To:             to,
		MissingCandles: missing,
	}
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
		Window: candleWindow(candles),
	}
	if len(candles) == 0 {
		result.Source = CandleSourceNone
		result.BaseInterval = ""
	}
	return result
}

func candleWindow(candles []Candle) CandleWindow {
	window := CandleWindow{Count: len(candles)}
	if len(candles) == 0 {
		return window
	}
	from := candles[0].OpenTime.UTC()
	to := candles[len(candles)-1].OpenTime.UTC()
	window.From = &from
	window.To = &to
	return window
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

func invalidCandleResult(
	query CandleQuery,
	source CandleSource,
	baseInterval string,
	candles []Candle,
	validationInterval string,
	code string,
	err error,
) CandleResult {
	result := CandleResult{
		Source:            source,
		RequestedInterval: query.Interval,
		BaseInterval:      baseInterval,
		Health:            CandleHealthInvalid,
		Issues:            []CandleIssue{firstCandleIssue(candles, validationInterval, code, err)},
		Coverage: CandleCoverage{
			RequestedLimit: NormalizeCandleLimit(query.Limit),
		},
	}
	if source == CandleSourceNone {
		result.BaseInterval = ""
	}
	return result
}

func firstCandleIssue(candles []Candle, interval string, code string, err error) CandleIssue {
	issue := CandleIssue{Code: code, Message: err.Error()}
	for _, candle := range candles {
		if valueErr := validateCandleValues(candle); valueErr != nil {
			openTime := candle.OpenTime.UTC()
			issue.Message = valueErr.Error()
			issue.OpenTime = &openTime
			return issue
		}
	}
	for _, candle := range candles {
		if singleErr := validateCandleSeries([]Candle{candle}, interval); singleErr != nil {
			openTime := candle.OpenTime.UTC()
			issue.Message = singleErr.Error()
			issue.OpenTime = &openTime
			return issue
		}
	}
	return issue
}

func (provider *CandleProvider) withPagination(
	ctx context.Context,
	query CandleQuery,
	result CandleResult,
) (CandleResult, error) {
	pagination, err := provider.pagination(ctx, query, result)
	if err != nil {
		return CandleResult{}, err
	}
	result.Pagination = pagination
	return result, nil
}

func (provider *CandleProvider) pagination(
	ctx context.Context,
	query CandleQuery,
	result CandleResult,
) (CandlePagination, error) {
	if len(result.Candles) == 0 {
		return CandlePagination{}, nil
	}

	intervalDuration, err := IntervalDuration(query.Interval)
	if err != nil {
		return CandlePagination{}, err
	}
	limit := NormalizeCandleLimit(query.Limit)
	firstOpen := result.Candles[0].OpenTime.UTC()
	lastOpen := result.Candles[len(result.Candles)-1].OpenTime.UTC()

	previousTo := firstOpen.Add(-intervalDuration)
	previousFrom := previousTo.Add(-time.Duration(limit-1) * intervalDuration)
	nextFrom := lastOpen.Add(intervalDuration)
	nextTo := nextFrom.Add(time.Duration(limit-1) * intervalDuration)

	probeInterval := query.Interval
	probePreviousTo := previousTo
	probeNextFrom := nextFrom
	if result.Source == CandleSourceAggregated {
		baseDuration, err := IntervalDuration(baseCandleInterval)
		if err != nil {
			return CandlePagination{}, err
		}
		probeInterval = baseCandleInterval
		probePreviousTo = firstOpen.Add(-baseDuration)
	}

	hasPrevious, err := provider.hasNativeCandle(ctx, query, probeInterval, nil, &probePreviousTo)
	if err != nil {
		return CandlePagination{}, err
	}
	hasNext, err := provider.hasNativeCandle(ctx, query, probeInterval, &probeNextFrom, nil)
	if err != nil {
		return CandlePagination{}, err
	}

	pagination := CandlePagination{
		HasPrevious: hasPrevious,
		HasNext:     hasNext,
	}
	if hasPrevious {
		pagination.PreviousFrom = &previousFrom
		pagination.PreviousTo = &previousTo
		cursor, err := EncodeCandleCursor(NewCandleCursor(query, previousFrom, previousTo, limit))
		if err != nil {
			return CandlePagination{}, err
		}
		pagination.PreviousCursor = cursor
	}
	if hasNext {
		pagination.NextFrom = &nextFrom
		pagination.NextTo = &nextTo
		cursor, err := EncodeCandleCursor(NewCandleCursor(query, nextFrom, nextTo, limit))
		if err != nil {
			return CandlePagination{}, err
		}
		pagination.NextCursor = cursor
	}
	return pagination, nil
}

func (provider *CandleProvider) hasNativeCandle(
	ctx context.Context,
	query CandleQuery,
	interval string,
	from *time.Time,
	to *time.Time,
) (bool, error) {
	probeQuery := CandleQuery{
		Exchange: query.Exchange,
		Symbol:   query.Symbol,
		Interval: interval,
		From:     from,
		To:       to,
		Limit:    1,
	}
	candles, err := provider.store.ListNativeCandles(ctx, probeQuery)
	if err != nil {
		return false, err
	}
	return len(candles) > 0, nil
}

type aggregationBaseWindow struct {
	Required int
	Limit    int
	Limited  bool
}
