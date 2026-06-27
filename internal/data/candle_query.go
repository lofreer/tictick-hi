package data

import (
	"fmt"
	"time"
)

const (
	DefaultCandleLimit = 1000
	MaxCandleLimit     = 5000
)

func NormalizeCandleLimit(limit int) int {
	if limit <= 0 {
		return DefaultCandleLimit
	}
	if limit > MaxCandleLimit {
		return MaxCandleLimit
	}
	return limit
}

func ValidateCandleQueryRange(query CandleQuery) error {
	intervalDuration, err := IntervalDuration(query.Interval)
	if err != nil {
		return err
	}

	if query.From == nil || query.To == nil {
		return nil
	}
	if query.To.Before(*query.From) {
		return fmt.Errorf("from must be before or equal to to")
	}

	maxSpan := MaxCandleQuerySpan(intervalDuration)
	if query.To.Sub(*query.From) > maxSpan {
		return fmt.Errorf("time range must cover at most %d candles for interval %s", MaxCandleLimit, query.Interval)
	}
	return nil
}

func MaxCandleQuerySpan(intervalDuration time.Duration) time.Duration {
	return intervalDuration * time.Duration(MaxCandleLimit-1)
}
