package data

import (
	"fmt"
	"math/big"
	"time"
)

func ValidateCandleSeries(candles []Candle, interval string) error {
	return validateCandleSeries(candles, interval)
}

func ValidateCandleSeriesForTarget(candles []Candle, exchange string, symbol string, interval string) error {
	for _, candle := range candles {
		if candle.Exchange != exchange || candle.Symbol != symbol || candle.Interval != interval {
			return fmt.Errorf(
				"candle %s target does not match data sync task %s/%s/%s",
				candleIdentity(candle),
				exchange,
				symbol,
				interval,
			)
		}
	}
	return validateCandleSeries(candles, interval)
}

func DetectCandleIssue(candle Candle) *CandleIssue {
	open, err := parseCandleDecimal(candle, "open", candle.Open)
	if err != nil {
		return candleIssue(candle, CandleIssueInvalidOpenPrice, err.Error())
	}
	high, err := parseCandleDecimal(candle, "high", candle.High)
	if err != nil {
		return candleIssue(candle, CandleIssueInvalidHighPrice, err.Error())
	}
	low, err := parseCandleDecimal(candle, "low", candle.Low)
	if err != nil {
		return candleIssue(candle, CandleIssueInvalidLowPrice, err.Error())
	}
	closeValue, err := parseCandleDecimal(candle, "close", candle.Close)
	if err != nil {
		return candleIssue(candle, CandleIssueInvalidClosePrice, err.Error())
	}
	volume, err := parseCandleDecimal(candle, "volume", candle.Volume)
	if err != nil {
		return candleIssue(candle, CandleIssueInvalidVolume, err.Error())
	}

	zero := new(big.Rat)
	if open.Cmp(zero) <= 0 {
		return candleIssue(candle, CandleIssueInvalidOpenPrice, "open price value must be positive")
	}
	if high.Cmp(zero) <= 0 {
		return candleIssue(candle, CandleIssueInvalidHighPrice, "high price value must be positive")
	}
	if low.Cmp(zero) <= 0 {
		return candleIssue(candle, CandleIssueInvalidLowPrice, "low price value must be positive")
	}
	if closeValue.Cmp(zero) <= 0 {
		return candleIssue(candle, CandleIssueInvalidClosePrice, "close price value must be positive")
	}
	if volume.Cmp(zero) < 0 {
		return candleIssue(candle, CandleIssueInvalidVolume, "volume value is negative")
	}
	if high.Cmp(open) < 0 || high.Cmp(closeValue) < 0 || high.Cmp(low) < 0 {
		return candleIssue(candle, CandleIssueInvalidHighBound, "high value is below OHLC bounds")
	}
	if low.Cmp(open) > 0 || low.Cmp(closeValue) > 0 || low.Cmp(high) > 0 {
		return candleIssue(candle, CandleIssueInvalidLowBound, "low value is above OHLC bounds")
	}
	if issue := detectCandleTimeIssue(candle, candle.Interval); issue != nil {
		return issue
	}
	return nil
}

func detectCandleTimeIssue(candle Candle, interval string) *CandleIssue {
	duration, err := IntervalDuration(interval)
	if err != nil {
		return nil
	}
	openTime := candle.OpenTime.UTC()
	if !alignTime(openTime, duration).Equal(openTime) {
		return candleIssue(
			candle,
			CandleIssueInvalidOpenTime,
			fmt.Sprintf("open time %s is not aligned to interval %s", openTime.Format(time.RFC3339), interval),
		)
	}
	expectedClose := openTime.Add(duration)
	closeTime := candle.CloseTime.UTC()
	if !closeTime.Equal(expectedClose) {
		return candleIssue(
			candle,
			CandleIssueInvalidCloseTime,
			fmt.Sprintf("close time %s does not match expected %s for interval %s", closeTime.Format(time.RFC3339), expectedClose.Format(time.RFC3339), interval),
		)
	}
	return nil
}

func validateCandleSeries(candles []Candle, interval string) error {
	duration, err := IntervalDuration(interval)
	if err != nil {
		return err
	}

	var previousOpen time.Time
	for index, candle := range candles {
		openTime := candle.OpenTime.UTC()
		closeTime := candle.CloseTime.UTC()
		if err := validateCandleValues(candle); err != nil {
			return err
		}
		if !alignTime(openTime, duration).Equal(openTime) {
			return fmt.Errorf("candle %s open time %s is not aligned to interval %s", candleIdentity(candle), openTime.Format(time.RFC3339), interval)
		}
		expectedClose := openTime.Add(duration)
		if !closeTime.Equal(expectedClose) {
			return fmt.Errorf("candle %s close time %s does not match expected %s for interval %s", candleIdentity(candle), closeTime.Format(time.RFC3339), expectedClose.Format(time.RFC3339), interval)
		}
		if index > 0 {
			if openTime.Before(previousOpen) {
				return newCandleSeriesIssueError(candle, fmt.Sprintf("candle %s open time %s is out of order", candleIdentity(candle), openTime.Format(time.RFC3339)))
			}
			if openTime.Equal(previousOpen) {
				return newCandleSeriesIssueError(candle, fmt.Sprintf("candle %s has duplicate open time %s", candleIdentity(candle), openTime.Format(time.RFC3339)))
			}
		}
		previousOpen = openTime
	}
	return nil
}

type candleSeriesIssueError struct {
	message  string
	openTime time.Time
}

func newCandleSeriesIssueError(candle Candle, message string) candleSeriesIssueError {
	return candleSeriesIssueError{message: message, openTime: candle.OpenTime.UTC()}
}

func (err candleSeriesIssueError) Error() string {
	return err.message
}

func validateCandleValues(candle Candle) error {
	open, err := parseCandleDecimal(candle, "open", candle.Open)
	if err != nil {
		return err
	}
	high, err := parseCandleDecimal(candle, "high", candle.High)
	if err != nil {
		return err
	}
	low, err := parseCandleDecimal(candle, "low", candle.Low)
	if err != nil {
		return err
	}
	closeValue, err := parseCandleDecimal(candle, "close", candle.Close)
	if err != nil {
		return err
	}
	volume, err := parseCandleDecimal(candle, "volume", candle.Volume)
	if err != nil {
		return err
	}

	zero := new(big.Rat)
	for _, item := range []struct {
		field string
		value *big.Rat
	}{
		{field: "open", value: open},
		{field: "high", value: high},
		{field: "low", value: low},
		{field: "close", value: closeValue},
	} {
		if item.value.Cmp(zero) <= 0 {
			return fmt.Errorf("candle %s %s price value must be positive", candleIdentity(candle), item.field)
		}
	}
	if volume.Cmp(zero) < 0 {
		return fmt.Errorf("candle %s volume value is negative", candleIdentity(candle))
	}

	if high.Cmp(open) < 0 || high.Cmp(closeValue) < 0 || high.Cmp(low) < 0 {
		return fmt.Errorf("candle %s high value is below OHLC bounds", candleIdentity(candle))
	}
	if low.Cmp(open) > 0 || low.Cmp(closeValue) > 0 || low.Cmp(high) > 0 {
		return fmt.Errorf("candle %s low value is above OHLC bounds", candleIdentity(candle))
	}
	return nil
}

func parseCandleDecimal(candle Candle, field string, value string) (*big.Rat, error) {
	parsed, err := parseDecimal(value)
	if err != nil {
		return nil, fmt.Errorf("candle %s %s %q is not a decimal", candleIdentity(candle), field, value)
	}
	return parsed, nil
}

func candleIssue(candle Candle, code string, message string) *CandleIssue {
	openTime := candle.OpenTime.UTC()
	return &CandleIssue{
		Code:     code,
		Message:  message,
		OpenTime: &openTime,
	}
}

func candleIdentity(candle Candle) string {
	return fmt.Sprintf("%s/%s/%s", candle.Exchange, candle.Symbol, candle.Interval)
}
