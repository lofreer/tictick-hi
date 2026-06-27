package data

import "fmt"

func ValidateStrategyCandleResult(result CandleResult) error {
	if result.Health != CandleHealthOK {
		return fmt.Errorf("candle data health is %s", result.Health)
	}
	if result.Coverage.LimitedByBaseWindow {
		return fmt.Errorf(
			"candle data coverage is limited by base window: returned %d of %d requested candles",
			result.Coverage.ReturnedCandles,
			result.Coverage.RequestedLimit,
		)
	}
	return nil
}
