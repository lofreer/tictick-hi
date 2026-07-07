package api

import (
	"context"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) CheckLoginRateLimit(
	_ context.Context,
	keyHash string,
	now time.Time,
	window time.Duration,
) (bool, error) {
	state, exists := repository.loginRateLimits[keyHash]
	if !exists {
		return true, nil
	}
	if state.LockedUntil != nil && state.LockedUntil.After(now) {
		return false, nil
	}
	if window > 0 && now.Sub(state.FirstFailureAt) > window {
		delete(repository.loginRateLimits, keyHash)
	}
	return true, nil
}

func (repository *fakeRepository) RecordLoginFailure(
	_ context.Context,
	keyHash string,
	now time.Time,
	limit int,
	window time.Duration,
	lockout time.Duration,
) error {
	if repository.loginRateLimits == nil {
		repository.loginRateLimits = map[string]data.LoginRateLimitState{}
	}
	state, exists := repository.loginRateLimits[keyHash]
	if !exists || (window > 0 && now.Sub(state.FirstFailureAt) > window) {
		state = data.LoginRateLimitState{
			KeyHash:        keyHash,
			FirstFailureAt: now,
		}
	}
	state.FailureCount++
	state.LockedUntil = nil
	if limit > 0 && state.FailureCount >= limit {
		lockedUntil := now.Add(lockout)
		state.LockedUntil = &lockedUntil
	}
	state.UpdatedAt = now
	repository.loginRateLimits[keyHash] = state
	return nil
}

func (repository *fakeRepository) ClearLoginRateLimit(_ context.Context, keyHash string) error {
	delete(repository.loginRateLimits, keyHash)
	return nil
}
