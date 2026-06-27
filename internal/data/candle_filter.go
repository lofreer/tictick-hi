package data

func ClosedCandles(candles []Candle) []Candle {
	result := make([]Candle, 0, len(candles))
	for _, candle := range candles {
		if candle.IsClosed {
			result = append(result, candle)
		}
	}
	return result
}
