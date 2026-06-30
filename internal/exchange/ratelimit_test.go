package exchange

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestFixedWindowRateLimiterWaitsForNextWindow(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(1, 20*time.Millisecond)

	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
		t.Fatalf("second reservation waited %s, want at least 15ms", elapsed)
	}
}

func TestFixedWindowRateLimiterHonorsContextCancellation(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(1, time.Hour)
	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err := limiter.Wait(ctx, 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error = %v, want deadline exceeded", err)
	}
}

func TestFixedWindowRateLimiterRejectsOverweightRequest(t *testing.T) {
	limiter := NewFixedWindowRateLimiter(2, time.Minute)

	err := limiter.Wait(context.Background(), 3)
	if err == nil {
		t.Fatal("expected overweight request to fail")
	}
}

func TestMultiWindowRateLimiterWaitsForRestrictiveWindow(t *testing.T) {
	limiter, err := NewMultiWindowRateLimiter([]RateLimitWindow{
		{Limit: 1, Window: 20 * time.Millisecond},
		{Limit: 10, Window: time.Minute},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}
	if elapsed := time.Since(started); elapsed < 15*time.Millisecond {
		t.Fatalf("second reservation waited %s, want at least 15ms", elapsed)
	}
}

func TestMultiWindowRateLimiterRejectsOverweightRequest(t *testing.T) {
	limiter, err := NewMultiWindowRateLimiter([]RateLimitWindow{
		{Limit: 5, Window: time.Minute},
		{Limit: 2, Window: time.Second},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = limiter.Wait(context.Background(), 3)
	if err == nil {
		t.Fatal("expected overweight request to fail")
	}
}

func TestMultiWindowRateLimiterDoesNotPartiallyReserveOnBlockedWindow(t *testing.T) {
	limiter, err := NewMultiWindowRateLimiter([]RateLimitWindow{
		{Limit: 2, Window: time.Hour},
		{Limit: 1, Window: time.Hour},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err = limiter.Wait(ctx, 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error = %v, want deadline exceeded", err)
	}

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	for _, window := range limiter.windows {
		if window.used != 1 {
			t.Fatalf("window used = %d, want 1 after blocked reservation", window.used)
		}
	}
}

func TestMultiWindowRateLimiterHonorsInitialUsage(t *testing.T) {
	limiter, err := NewMultiWindowRateLimiterWithInitialUsage([]RateLimitWindow{
		{Limit: 3, Window: time.Hour},
		{Limit: 5, Window: time.Hour},
	}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := limiter.Wait(context.Background(), 1); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	err = limiter.Wait(ctx, 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Wait error = %v, want deadline exceeded", err)
	}
}
