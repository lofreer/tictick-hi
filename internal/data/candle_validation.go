package data

import (
	"fmt"
	"time"
)

func validateCandleSeries(candles []Candle, interval string) error {
	duration, err := IntervalDuration(interval)
	if err != nil {
		return err
	}

	var previousOpen time.Time
	for index, candle := range candles {
		openTime := candle.OpenTime.UTC()
		closeTime := candle.CloseTime.UTC()
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

func candleIdentity(candle Candle) string {
	return fmt.Sprintf("%s/%s/%s", candle.Exchange, candle.Symbol, candle.Interval)
}
