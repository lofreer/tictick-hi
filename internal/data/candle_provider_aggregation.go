package data

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultMaxAggregationBasePages = 288
)

var maxAggregationBasePages = defaultMaxAggregationBasePages

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
	maxAggregationBaseCandles := MaxCandleLimit * maxAggregationBasePages
	if required > maxAggregationBaseCandles {
		return aggregationBaseWindow{Required: required, Limit: maxAggregationBaseCandles, Limited: true}
	}
	return aggregationBaseWindow{Required: required, Limit: required}
}

func (provider *CandleProvider) aggregateBaseCandles(
	ctx context.Context,
	query CandleQuery,
	window aggregationBaseWindow,
	targetInterval string,
) (aggregationBuildResult, error) {
	if window.Limit <= 0 {
		return aggregationBuildResult{}, nil
	}
	if query.From == nil {
		streamQuery, err := provider.latestAggregationStreamQuery(ctx, query, window.Limit)
		if err != nil || streamQuery.From == nil {
			return aggregationBuildResult{}, err
		}
		query = streamQuery
	}
	return provider.aggregateForwardBaseCandles(ctx, query, window.Limit, targetInterval)
}

func (provider *CandleProvider) latestAggregationStreamQuery(
	ctx context.Context,
	query CandleQuery,
	limit int,
) (CandleQuery, error) {
	var to *time.Time
	if query.To != nil {
		value := query.To.UTC()
		to = &value
	}

	returned := 0
	var earliestOpen *time.Time
	var latestOpen *time.Time
	for returned < limit {
		pageLimit := aggregationPageLimit(limit - returned)
		pageQuery := query
		pageQuery.To = to
		pageQuery.Limit = pageLimit

		page, err := provider.store.ListLatestNativeCandles(ctx, pageQuery)
		if err != nil {
			return CandleQuery{}, err
		}
		if len(page) == 0 {
			break
		}

		if latestOpen == nil {
			value := page[len(page)-1].OpenTime.UTC()
			latestOpen = &value
		}
		value := page[0].OpenTime.UTC()
		earliestOpen = &value
		returned += len(page)
		if len(page) < pageLimit {
			break
		}

		previousTo := page[0].OpenTime.UTC().Add(-time.Minute)
		to = &previousTo
	}

	streamQuery := query
	streamQuery.From = earliestOpen
	if streamQuery.To == nil {
		streamQuery.To = latestOpen
	}
	streamQuery.Limit = limit
	return streamQuery, nil
}

func (provider *CandleProvider) aggregateForwardBaseCandles(
	ctx context.Context,
	query CandleQuery,
	limit int,
	targetInterval string,
) (aggregationBuildResult, error) {
	if query.From == nil {
		return aggregationBuildResult{}, nil
	}

	from := query.From.UTC()
	result := aggregationBuildResult{}
	gapTracker := newCandleGapTracker(baseCandleInterval, query.From, query.To)
	aggregator := newCandleAggregator(targetInterval)
	var previousOpen *time.Time

	for result.ReturnedBaseCandles < limit {
		pageLimit := aggregationPageLimit(limit - result.ReturnedBaseCandles)
		pageQuery := query
		pageQuery.From = &from
		pageQuery.Limit = pageLimit

		page, err := provider.store.ListNativeCandles(ctx, pageQuery)
		if err != nil {
			return aggregationBuildResult{}, err
		}
		if len(page) == 0 {
			break
		}

		if err := validateAggregationBasePage(page, previousOpen); err != nil {
			result.InvalidCandles = page
			result.InvalidErr = err
			return result, nil
		}

		for _, candle := range page {
			gapTracker.Add(candle)
			if err := aggregator.Add(candle); err != nil {
				result.InvalidCandles = page
				result.InvalidErr = err
				return result, nil
			}
		}
		lastOpen := page[len(page)-1].OpenTime.UTC()
		previousOpen = &lastOpen
		result.ReturnedBaseCandles += len(page)
		if len(page) < pageLimit {
			break
		}

		nextFrom := lastOpen.Add(time.Minute)
		if !nextFrom.After(from) || query.To != nil && nextFrom.After(query.To.UTC()) {
			break
		}
		from = nextFrom
	}
	result.Candles = aggregator.Flush()
	result.Gaps = gapTracker.Finish()
	return result, nil
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

type aggregationBuildResult struct {
	Candles             []Candle
	Gaps                []CandleGap
	ReturnedBaseCandles int
	InvalidCandles      []Candle
	InvalidErr          error
}

type streamingCandleAggregator struct {
	inner candleAggregator
}

func newCandleAggregator(targetInterval string) *streamingCandleAggregator {
	targetDuration, _ := IntervalDuration(targetInterval)
	baseDuration, _ := IntervalDuration(baseCandleInterval)
	return &streamingCandleAggregator{
		inner: candleAggregator{
			targetInterval: targetInterval,
			targetDuration: targetDuration,
			baseDuration:   baseDuration,
			expectedCount:  int(targetDuration / baseDuration),
		},
	}
}

func (aggregator *streamingCandleAggregator) Add(candle Candle) error {
	return aggregator.inner.add(candle)
}

func (aggregator *streamingCandleAggregator) Flush() []Candle {
	aggregator.inner.flush()
	return aggregator.inner.result
}

func validateAggregationBasePage(page []Candle, previousOpen *time.Time) error {
	if err := validateCandleSeries(page, baseCandleInterval); err != nil {
		return err
	}
	if previousOpen == nil || len(page) == 0 {
		return nil
	}
	firstOpen := page[0].OpenTime.UTC()
	if firstOpen.Before(*previousOpen) {
		return fmt.Errorf("candle %s open time %s is out of order", candleIdentity(page[0]), firstOpen.Format(time.RFC3339))
	}
	if firstOpen.Equal(*previousOpen) {
		return fmt.Errorf("candle %s has duplicate open time %s", candleIdentity(page[0]), firstOpen.Format(time.RFC3339))
	}
	return nil
}

type candleGapTracker struct {
	duration     time.Duration
	from         *time.Time
	to           *time.Time
	previousOpen *time.Time
	gaps         []CandleGap
}

func newCandleGapTracker(interval string, from *time.Time, to *time.Time) *candleGapTracker {
	duration, _ := IntervalDuration(interval)
	return &candleGapTracker{duration: duration, from: from, to: to}
}

func (tracker *candleGapTracker) Add(candle Candle) {
	openTime := candle.OpenTime.UTC()
	if tracker.previousOpen == nil {
		if tracker.from != nil {
			firstExpected := firstExpectedOpen(*tracker.from, tracker.duration)
			if openTime.After(firstExpected) {
				tracker.gaps = append(tracker.gaps, newCandleGap(firstExpected, openTime, tracker.duration))
			}
		}
		tracker.previousOpen = &openTime
		return
	}

	expected := tracker.previousOpen.Add(tracker.duration)
	if openTime.After(expected) {
		tracker.gaps = append(tracker.gaps, newCandleGap(expected, openTime, tracker.duration))
	}
	tracker.previousOpen = &openTime
}

func (tracker *candleGapTracker) Finish() []CandleGap {
	if tracker.previousOpen == nil {
		if tracker.from == nil || tracker.to == nil {
			return tracker.gaps
		}
		firstExpected := firstExpectedOpen(*tracker.from, tracker.duration)
		lastExpected := alignTime(*tracker.to, tracker.duration)
		if lastExpected.Before(firstExpected) {
			return tracker.gaps
		}
		return append(tracker.gaps, newCandleGap(firstExpected, lastExpected.Add(tracker.duration), tracker.duration))
	}
	if tracker.to != nil {
		lastExpected := alignTime(*tracker.to, tracker.duration)
		nextExpected := tracker.previousOpen.Add(tracker.duration)
		if !lastExpected.Before(nextExpected) {
			tracker.gaps = append(tracker.gaps, newCandleGap(nextExpected, lastExpected.Add(tracker.duration), tracker.duration))
		}
	}
	return tracker.gaps
}
