package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
)

func (store *Store) tryExchangeAdvisoryLock(
	ctx context.Context,
	keyPrefix string,
	exchangeID string,
	label string,
) (func(context.Context) error, bool, error) {
	lockKey, normalizedExchange, err := exchangeAdvisoryLockKey(keyPrefix, exchangeID, label)
	if err != nil {
		return nil, false, err
	}
	conn, err := store.pool.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire %s lock connection: %w", label, err)
	}

	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		conn.Release()
		return nil, false, fmt.Errorf("acquire %s lock: %w", label, err)
	}
	if !locked {
		conn.Release()
		return nil, false, nil
	}

	released := false
	return func(ctx context.Context) error {
		if released {
			return nil
		}
		released = true

		var unlocked bool
		if err := conn.QueryRow(ctx, `SELECT pg_advisory_unlock($1)`, lockKey).Scan(&unlocked); err != nil {
			pgConn := conn.Hijack()
			if closeErr := pgConn.Close(ctx); closeErr != nil {
				return fmt.Errorf("release %s lock: %w; close locked connection: %v", label, err, closeErr)
			}
			return fmt.Errorf("release %s lock: %w", label, err)
		}
		conn.Release()
		if !unlocked {
			return fmt.Errorf("%s lock was not held for %s", label, normalizedExchange)
		}
		return nil
	}, true, nil
}

func exchangeAdvisoryLockKey(keyPrefix string, exchangeID string, label string) (int64, string, error) {
	exchangeID = strings.ToLower(strings.TrimSpace(exchangeID))
	if exchangeID == "" {
		return 0, "", fmt.Errorf("%s lock exchange is required", label)
	}
	digest := sha256.Sum256([]byte(keyPrefix + exchangeID))
	return int64(binary.BigEndian.Uint64(digest[:8])), exchangeID, nil
}
