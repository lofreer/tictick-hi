package data

import "time"

type LoginRateLimitState struct {
	KeyHash        string
	FailureCount   int
	FirstFailureAt time.Time
	LockedUntil    *time.Time
	UpdatedAt      time.Time
}
