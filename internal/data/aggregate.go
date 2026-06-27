package data

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
)

func AggregateCandles(baseCandles []Candle, targetInterval string) ([]Candle, error) {
	targetDuration, err := IntervalDuration(targetInterval)
	if err != nil {
		return nil, err
	}
	baseDuration, err := IntervalDuration("1m")
	if err != nil {
		return nil, err
	}
	if targetDuration <= baseDuration || targetDuration%baseDuration != 0 {
		return nil, fmt.Errorf("target interval %q cannot be aggregated from 1m", targetInterval)
	}
	if len(baseCandles) == 0 {
		return nil, nil
	}

	aggregator := candleAggregator{
		targetInterval: targetInterval,
		targetDuration: targetDuration,
		baseDuration:   baseDuration,
		expectedCount:  int(targetDuration / baseDuration),
	}
	for _, candle := range sortedCandles(baseCandles) {
		if err := aggregator.add(candle); err != nil {
			return nil, err
		}
	}
	aggregator.flush()

	return aggregator.result, nil
}

func sortedCandles(candles []Candle) []Candle {
	ordered := append([]Candle(nil), candles...)
	sort.Slice(ordered, func(i int, j int) bool {
		return ordered[i].OpenTime.Before(ordered[j].OpenTime)
	})
	return ordered
}

type candleAggregator struct {
	targetInterval string
	targetDuration time.Duration
	baseDuration   time.Duration
	expectedCount  int
	current        *aggregateWindow
	result         []Candle
}

type aggregateWindow struct {
	candle       Candle
	count        int
	volume       *big.Rat
	lastOpenTime time.Time
	hasGap       bool
	allClosed    bool
}

func (aggregator *candleAggregator) add(candle Candle) error {
	windowStart := alignTime(candle.OpenTime, aggregator.targetDuration)
	if aggregator.current == nil || !aggregator.current.candle.OpenTime.Equal(windowStart) {
		aggregator.flush()
		window, err := newWindow(candle, windowStart, aggregator.targetDuration, aggregator.targetInterval)
		if err != nil {
			return err
		}
		aggregator.current = window
		return nil
	}

	if !candle.OpenTime.Equal(aggregator.current.lastOpenTime.Add(aggregator.baseDuration)) {
		aggregator.current.hasGap = true
	}
	aggregator.current.lastOpenTime = candle.OpenTime
	aggregator.current.count++
	aggregator.current.candle.Close = candle.Close
	aggregator.current.allClosed = aggregator.current.allClosed && candle.IsClosed

	if compareDecimal(candle.High, aggregator.current.candle.High) > 0 {
		aggregator.current.candle.High = candle.High
	}
	if compareDecimal(candle.Low, aggregator.current.candle.Low) < 0 {
		aggregator.current.candle.Low = candle.Low
	}

	volume, err := parseDecimal(candle.Volume)
	if err != nil {
		return err
	}
	aggregator.current.volume.Add(aggregator.current.volume, volume)
	aggregator.current.candle.Volume = formatDecimal(aggregator.current.volume)
	return nil
}

func (aggregator *candleAggregator) flush() {
	if aggregator.current == nil {
		return
	}
	if !aggregator.current.hasGap {
		aggregator.current.candle.IsClosed = aggregator.current.count == aggregator.expectedCount &&
			aggregator.current.allClosed
		aggregator.result = append(aggregator.result, aggregator.current.candle)
	}
	aggregator.current = nil
}

func newWindow(
	candle Candle,
	windowStart time.Time,
	targetDuration time.Duration,
	targetInterval string,
) (*aggregateWindow, error) {
	volume, err := parseDecimal(candle.Volume)
	if err != nil {
		return nil, err
	}
	return &aggregateWindow{
		candle: Candle{
			Exchange:  candle.Exchange,
			Symbol:    candle.Symbol,
			Interval:  targetInterval,
			OpenTime:  windowStart,
			CloseTime: windowStart.Add(targetDuration),
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
			IsClosed:  false,
		},
		count:        1,
		volume:       volume,
		lastOpenTime: candle.OpenTime,
		allClosed:    candle.IsClosed,
	}, nil
}

func alignTime(value time.Time, duration time.Duration) time.Time {
	utc := value.UTC()
	return time.Unix(0, utc.UnixNano()/duration.Nanoseconds()*duration.Nanoseconds()).UTC()
}

func compareDecimal(left string, right string) int {
	leftValue, leftErr := parseDecimal(left)
	rightValue, rightErr := parseDecimal(right)
	if leftErr != nil || rightErr != nil {
		return strings.Compare(left, right)
	}
	return leftValue.Cmp(rightValue)
}

func parseDecimal(value string) (*big.Rat, error) {
	parsed := new(big.Rat)
	if _, ok := parsed.SetString(value); !ok {
		return nil, fmt.Errorf("invalid decimal %q", value)
	}
	return parsed, nil
}

func formatDecimal(value *big.Rat) string {
	text := value.FloatString(12)
	text = strings.TrimRight(text, "0")
	text = strings.TrimRight(text, ".")
	if text == "" {
		return "0"
	}
	return text
}
