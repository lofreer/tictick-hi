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

type RateLimitWindow struct {
	Limit  int
	Window time.Duration
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

type MultiWindowRateLimiter struct {
	mu       sync.Mutex
	windows  []fixedWindowState
	now      func() time.Time
	newTimer func(time.Duration) *time.Timer
}

type fixedWindowState struct {
	limit   int
	window  time.Duration
	used    int
	resetAt time.Time
}

func NewMultiWindowRateLimiter(windows []RateLimitWindow) (*MultiWindowRateLimiter, error) {
	return NewMultiWindowRateLimiterWithInitialUsage(windows, 0)
}

func NewMultiWindowRateLimiterWithInitialUsage(
	windows []RateLimitWindow,
	initialUsage int,
) (*MultiWindowRateLimiter, error) {
	if len(windows) == 0 {
		return nil, fmt.Errorf("rate limit windows are not configured")
	}
	if initialUsage < 0 {
		initialUsage = 0
	}
	limiter := &MultiWindowRateLimiter{
		windows:  make([]fixedWindowState, 0, len(windows)),
		now:      time.Now,
		newTimer: time.NewTimer,
	}
	now := limiter.now()
	for _, window := range windows {
		if window.Limit <= 0 || window.Window <= 0 {
			return nil, fmt.Errorf("invalid rate limit window")
		}
		used := initialUsage
		if used > window.Limit {
			used = window.Limit
		}
		limiter.windows = append(limiter.windows, fixedWindowState{
			limit:   window.Limit,
			window:  window.Window,
			used:    used,
			resetAt: now.Add(window.Window),
		})
	}
	return limiter, nil
}

func (limiter *MultiWindowRateLimiter) Wait(ctx context.Context, weight int) error {
	if limiter == nil {
		return nil
	}
	if weight <= 0 {
		weight = 1
	}
	if err := limiter.validateWeight(weight); err != nil {
		return err
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

func (limiter *MultiWindowRateLimiter) validateWeight(weight int) error {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	if len(limiter.windows) == 0 {
		return fmt.Errorf("rate limit windows are not configured")
	}
	for _, window := range limiter.windows {
		if weight > window.limit {
			return fmt.Errorf("request weight %d exceeds rate limit capacity %d", weight, window.limit)
		}
	}
	return nil
}

func (limiter *MultiWindowRateLimiter) reserve(weight int) (time.Duration, bool) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	now := limiter.now()
	var waitFor time.Duration
	for index := range limiter.windows {
		window := &limiter.windows[index]
		if window.resetAt.IsZero() || !now.Before(window.resetAt) {
			window.used = 0
			window.resetAt = now.Add(window.window)
		}
		if window.used+weight > window.limit {
			if wait := window.resetAt.Sub(now); wait > waitFor {
				waitFor = wait
			}
		}
	}
	if waitFor > 0 {
		return waitFor, false
	}
	for index := range limiter.windows {
		limiter.windows[index].used += weight
	}
	return 0, true
}
