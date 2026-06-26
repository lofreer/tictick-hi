package strategy

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type Intent struct {
	ID         string
	Side       string
	Price      string
	Quantity   string
	OccurredAt time.Time
}

func GenerateIntents(
	_ context.Context,
	definition Definition,
	candles []data.Candle,
	params map[string]any,
) ([]Intent, error) {
	switch definition.ID {
	case "ema-cross":
		return emaCrossIntents(candles, params)
	case "breakout-range":
		return breakoutRangeIntents(candles, params)
	default:
		return nil, data.ErrNotFound
	}
}

func emaCrossIntents(candles []data.Candle, params map[string]any) ([]Intent, error) {
	if textParam(params, "signalMode", "order") != "order" {
		return nil, nil
	}

	fastPeriod := intParamValue(params, "fastPeriod", 12)
	slowPeriod := intParamValue(params, "slowPeriod", 26)
	if fastPeriod <= 0 || slowPeriod <= fastPeriod {
		return nil, errors.New("invalid ema periods")
	}

	closes, err := closePrices(candles)
	if err != nil {
		return nil, err
	}
	if len(closes) == 0 {
		return nil, nil
	}

	fast := closes[0]
	slow := closes[0]
	fastAlpha := 2 / (float64(fastPeriod) + 1)
	slowAlpha := 2 / (float64(slowPeriod) + 1)
	quantity := decimalParam(params, "orderSize", "0.01")

	var intents []Intent
	for index := 1; index < len(closes); index++ {
		previousFast := fast
		previousSlow := slow
		price := closes[index]
		fast = ema(price, fast, fastAlpha)
		slow = ema(price, slow, slowAlpha)

		switch {
		case previousFast <= previousSlow && fast > slow:
			intents = append(intents, intent("ema-cross", len(intents), "buy", candles[index], quantity))
		case previousFast >= previousSlow && fast < slow:
			intents = append(intents, intent("ema-cross", len(intents), "sell", candles[index], quantity))
		}
	}
	return intents, nil
}

func breakoutRangeIntents(candles []data.Candle, params map[string]any) ([]Intent, error) {
	lookback := intParamValue(params, "lookback", 20)
	if lookback <= 0 {
		return nil, errors.New("invalid lookback")
	}
	buffer := numberParamValue(params, "breakoutBufferPct", 0.2) / 100
	quantity := decimalParam(params, "orderSize", "0.01")
	side := textParam(params, "side", "both")

	var (
		intents  []Intent
		lastSide string
	)
	for index := lookback; index < len(candles); index++ {
		high, low, err := rangeHighLow(candles[index-lookback : index])
		if err != nil {
			return nil, err
		}
		closePrice, err := strconv.ParseFloat(candles[index].Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close price: %w", err)
		}

		if closePrice > high*(1+buffer) && side != "short" && lastSide != "buy" {
			intents = append(intents, intent("breakout-range", len(intents), "buy", candles[index], quantity))
			lastSide = "buy"
		}
		if closePrice < low*(1-buffer) && side != "long" && lastSide != "sell" {
			intents = append(intents, intent("breakout-range", len(intents), "sell", candles[index], quantity))
			lastSide = "sell"
		}
	}
	return intents, nil
}

func intent(prefix string, index int, side string, candle data.Candle, quantity string) Intent {
	return Intent{
		ID:         fmt.Sprintf("%s_%d", prefix, index+1),
		Side:       side,
		Price:      candle.Close,
		Quantity:   quantity,
		OccurredAt: candle.OpenTime,
	}
}

func closePrices(candles []data.Candle) ([]float64, error) {
	prices := make([]float64, 0, len(candles))
	for _, candle := range candles {
		price, err := strconv.ParseFloat(candle.Close, 64)
		if err != nil {
			return nil, fmt.Errorf("parse close price: %w", err)
		}
		prices = append(prices, price)
	}
	return prices, nil
}

func rangeHighLow(candles []data.Candle) (float64, float64, error) {
	if len(candles) == 0 {
		return 0, 0, errors.New("empty range")
	}
	high, err := strconv.ParseFloat(candles[0].High, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse high price: %w", err)
	}
	low, err := strconv.ParseFloat(candles[0].Low, 64)
	if err != nil {
		return 0, 0, fmt.Errorf("parse low price: %w", err)
	}
	for _, candle := range candles[1:] {
		nextHigh, err := strconv.ParseFloat(candle.High, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse high price: %w", err)
		}
		nextLow, err := strconv.ParseFloat(candle.Low, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("parse low price: %w", err)
		}
		if nextHigh > high {
			high = nextHigh
		}
		if nextLow < low {
			low = nextLow
		}
	}
	return high, low, nil
}

func ema(price float64, previous float64, alpha float64) float64 {
	return price*alpha + previous*(1-alpha)
}

func intParamValue(params map[string]any, key string, fallback int) int {
	return int(numberParamValue(params, key, float64(fallback)))
}

func numberParamValue(params map[string]any, key string, fallback float64) float64 {
	value, ok := params[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, err := strconv.ParseFloat(typed, 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func decimalParam(params map[string]any, key string, fallback string) string {
	value, ok := params[key]
	if !ok {
		return fallback
	}
	switch typed := value.(type) {
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(typed), 'f', -1, 32)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case string:
		return typed
	default:
		return fallback
	}
}

func textParam(params map[string]any, key string, fallback string) string {
	value, ok := params[key]
	if !ok {
		return fallback
	}
	text, ok := value.(string)
	if !ok || text == "" {
		return fallback
	}
	return text
}
