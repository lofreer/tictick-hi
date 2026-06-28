package exchange

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type RateLimiter interface {
	Wait(ctx context.Context, weight int) error
}

type FixedWindowRateLimiter struct {
	mu       sync.Mutex
	limit    int
	window   time.Duration
	used     int
	resetAt  time.Time
	now      func() time.Time
	newTimer func(time.Duration) *time.Timer
}

func NewFixedWindowRateLimiter(limit int, window time.Duration) *FixedWindowRateLimiter {
	limiter := &FixedWindowRateLimiter{
		limit:    limit,
		window:   window,
		now:      time.Now,
		newTimer: time.NewTimer,
	}
	limiter.resetAt = limiter.now().Add(window)
	return limiter
}

func (limiter *FixedWindowRateLimiter) Wait(ctx context.Context, weight int) error {
	if limiter == nil {
		return nil
	}
	if weight <= 0 {
		weight = 1
	}
	if limiter.limit <= 0 || limiter.window <= 0 {
		return fmt.Errorf("rate limit is not configured")
	}
	if weight > limiter.limit {
		return fmt.Errorf("request weight %d exceeds rate limit capacity %d", weight, limiter.limit)
	}

	for {
		waitFor, ok := limiter.reserve(weight)
		if ok {
			return nil
		}
		if waitFor <= 0 {
			continue
		}

		timer := limiter.newTimer(waitFor)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func (limiter *FixedWindowRateLimiter) reserve(weight int) (time.Duration, bool) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now()
	if limiter.resetAt.IsZero() || !now.Before(limiter.resetAt) {
		limiter.used = 0
		limiter.resetAt = now.Add(limiter.window)
	}

	if limiter.used+weight <= limiter.limit {
		limiter.used += weight
		return 0, true
	}
	return limiter.resetAt.Sub(now), false
}
