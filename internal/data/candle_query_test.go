package data

import (
	"testing"
	"time"
)

func TestNormalizeCandleLimit(t *testing.T) {
	cases := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "default", limit: 0, want: DefaultCandleLimit},
		{name: "negative", limit: -1, want: DefaultCandleLimit},
		{name: "requested", limit: 250, want: 250},
		{name: "max", limit: MaxCandleLimit, want: MaxCandleLimit},
		{name: "oversized", limit: MaxCandleLimit + 1, want: MaxCandleLimit},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeCandleLimit(tc.limit); got != tc.want {
				t.Fatalf("NormalizeCandleLimit(%d) = %d, want %d", tc.limit, got, tc.want)
			}
		})
	}
}

func TestValidateCandleQueryRange(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	maxOneMinuteSpan := MaxCandleQuerySpan(time.Minute)

	cases := []struct {
		name    string
		query   CandleQuery
		wantErr bool
	}{
		{
			name: "no bounds",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
			},
		},
		{
			name: "single bound",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				From:     &start,
			},
		},
		{
			name: "same bound",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				From:     &start,
				To:       &start,
			},
		},
		{
			name: "maximum span",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				From:     &start,
				To:       timePtr(start.Add(maxOneMinuteSpan)),
			},
		},
		{
			name: "inverted",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				From:     timePtr(start.Add(time.Minute)),
				To:       &start,
			},
			wantErr: true,
		},
		{
			name: "oversized",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "1m",
				From:     &start,
				To:       timePtr(start.Add(maxOneMinuteSpan + time.Minute)),
			},
			wantErr: true,
		},
		{
			name: "invalid interval",
			query: CandleQuery{
				Exchange: "binance",
				Symbol:   "BTCUSDT",
				Interval: "tick",
			},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCandleQueryRange(tc.query)
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
