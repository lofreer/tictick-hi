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
	if startTime != nil && endTime != nil && !startTime.Before(*endTime) {
		return fmt.Errorf("startTime must be before endTime")
	}
	return nil
}
