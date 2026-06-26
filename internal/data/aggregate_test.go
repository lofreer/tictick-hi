package data

import (
	"testing"
	"time"
)

func TestAggregateCandles(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := []Candle{
		testCandle(start, "10", "12", "9", "11", "1"),
		testCandle(start.Add(time.Minute), "11", "13", "10", "12", "2"),
		testCandle(start.Add(2*time.Minute), "12", "12.5", "8", "9", "3"),
		testCandle(start.Add(3*time.Minute), "9", "10", "7", "8", "4"),
		testCandle(start.Add(4*time.Minute), "8", "11", "8", "10", "5"),
	}

	aggregated, err := AggregateCandles(base, "5m")
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregated) != 1 {
		t.Fatalf("len = %d, want 1", len(aggregated))
	}

	candle := aggregated[0]
	if candle.Open != "10" || candle.High != "13" || candle.Low != "7" || candle.Close != "10" {
		t.Fatalf("unexpected ohlc: %#v", candle)
	}
	if candle.Volume != "15" || !candle.IsClosed {
		t.Fatalf("unexpected volume/closed: %#v", candle)
	}
}

func TestAggregateCandlesSkipsGappedWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	base := []Candle{
		testCandle(start, "10", "12", "9", "11", "1"),
		testCandle(start.Add(2*time.Minute), "12", "13", "11", "12", "1"),
	}

	aggregated, err := AggregateCandles(base, "5m")
	if err != nil {
		t.Fatal(err)
	}
	if len(aggregated) != 0 {
		t.Fatalf("aggregated len = %d, want 0", len(aggregated))
	}
}

func testCandle(openTime time.Time, open string, high string, low string, close string, volume string) Candle {
	return Candle{
		Exchange:  "binance",
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		OpenTime:  openTime,
		CloseTime: openTime.Add(time.Minute),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
		IsClosed:  true,
	}
}
