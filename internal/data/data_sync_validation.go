package data

import (
	"fmt"
	"time"
)

var supportedDataSyncIntervals = map[string]struct{}{
	"1m":  {},
	"5m":  {},
	"15m": {},
	"1h":  {},
	"4h":  {},
	"1d":  {},
}

func ValidateDataSyncTaskWindow(interval string, startTime *time.Time, endTime *time.Time) error {
	if _, ok := supportedDataSyncIntervals[interval]; !ok {
		return fmt.Errorf("unsupported data sync interval %q", interval)
	}
	duration, err := IntervalDuration(interval)
	if err != nil {
		return err
	}
	if startTime != nil && !alignedToInterval(*startTime, duration) {
		return fmt.Errorf("startTime must be aligned to %s interval", interval)
	}
	if endTime != nil && !alignedToInterval(*endTime, duration) {
		return fmt.Errorf("endTime must be aligned to %s interval", interval)
	}
	if startTime != nil && endTime != nil && !startTime.Before(*endTime) {
		return fmt.Errorf("startTime must be before endTime")
	}
	return nil
}

func alignedToInterval(value time.Time, interval time.Duration) bool {
	if interval <= 0 {
		return false
	}
	return value.UTC().UnixNano()%int64(interval) == 0
}
