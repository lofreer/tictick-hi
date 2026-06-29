package data

import (
	"fmt"
	"math/big"
	"time"
)

func ValidateCandleSeries(candles []Candle, interval string) error {
	return validateCandleSeries(candles, interval)
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
				return fmt.Errorf("candle %s open time %s is out of order", candleIdentity(candle), openTime.Format(time.RFC3339))
			}
			if openTime.Equal(previousOpen) {
				return fmt.Errorf("candle %s has duplicate open time %s", candleIdentity(candle), openTime.Format(time.RFC3339))
			}
		}
		previousOpen = openTime
	}
	return nil
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
		{field: "volume", value: volume},
	} {
		if item.value.Cmp(zero) < 0 {
			return fmt.Errorf("candle %s %s value is negative", candleIdentity(candle), item.field)
		}
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

func candleIdentity(candle Candle) string {
	return fmt.Sprintf("%s/%s/%s", candle.Exchange, candle.Symbol, candle.Interval)
}
