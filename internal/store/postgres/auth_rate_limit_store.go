package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

func (store *Store) CheckLoginRateLimit(
	ctx context.Context,
	keyHash string,
	now time.Time,
	window time.Duration,
) (bool, error) {
	var firstFailureAt time.Time
	var lockedUntil sql.NullTime
	err := store.pool.QueryRow(ctx, `
		SELECT first_failure_at, locked_until
		  FROM operator_login_rate_limits
		 WHERE key_hash = $1`,
		keyHash,
	).Scan(&firstFailureAt, &lockedUntil)
	if err == pgx.ErrNoRows {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("check login rate limit: %w", err)
	}

	if lockedUntil.Valid && lockedUntil.Time.After(now) {
		return false, nil
	}
	if window > 0 && now.Sub(firstFailureAt) > window {
		if err := store.ClearLoginRateLimit(ctx, keyHash); err != nil {
			return false, err
		}
	}
	return true, nil
}

func (store *Store) RecordLoginFailure(
	ctx context.Context,
	keyHash string,
	now time.Time,
	limit int,
	window time.Duration,
	lockout time.Duration,
) error {
	windowStart := time.Time{}
	if window > 0 {
		windowStart = now.Add(-window)
	}
	lockedUntil := now.Add(lockout)
	var initialLockedUntil any
	if limit <= 1 {
		initialLockedUntil = lockedUntil
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO operator_login_rate_limits (
			key_hash, failure_count, first_failure_at, locked_until, updated_at
		)
		VALUES ($1, 1, $2, $6, $2)
		ON CONFLICT (key_hash)
		DO UPDATE SET
			failure_count = CASE
				WHEN operator_login_rate_limits.first_failure_at < $4 THEN 1
				ELSE operator_login_rate_limits.failure_count + 1
			END,
			first_failure_at = CASE
				WHEN operator_login_rate_limits.first_failure_at < $4 THEN $2
				ELSE operator_login_rate_limits.first_failure_at
			END,
			locked_until = CASE
				WHEN (
					CASE
						WHEN operator_login_rate_limits.first_failure_at < $4 THEN 1
						ELSE operator_login_rate_limits.failure_count + 1
					END
				) >= $3 THEN $5
				ELSE NULL
			END,
			updated_at = $2`,
		keyHash,
		now,
		limit,
		windowStart,
		lockedUntil,
		initialLockedUntil,
	); err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	return nil
}

func (store *Store) ClearLoginRateLimit(ctx context.Context, keyHash string) error {
	if _, err := store.pool.Exec(ctx, `
		DELETE FROM operator_login_rate_limits
		 WHERE key_hash = $1`,
		keyHash,
	); err != nil {
		return fmt.Errorf("clear login rate limit: %w", err)
	}
	return nil
}
