package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"
)

const marketInstrumentSyncLockKeyPrefix = "tictick-hi:market-instrument-sync:"

func (store *Store) TryLockMarketInstrumentSync(
	ctx context.Context,
	exchangeID string,
) (func(context.Context) error, bool, error) {
	lockKey, err := marketInstrumentSyncLockKey(exchangeID)
	if err != nil {
		return nil, false, err
	}
	conn, err := store.pool.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("acquire market instrument sync lock connection: %w", err)
	}

	var locked bool
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, lockKey).Scan(&locked); err != nil {
		conn.Release()
		return nil, false, fmt.Errorf("acquire market instrument sync lock: %w", err)
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
				return fmt.Errorf(
					"release market instrument sync lock: %w; close locked connection: %v",
					err,
					closeErr,
				)
			}
			return fmt.Errorf("release market instrument sync lock: %w", err)
		}
		conn.Release()
		if !unlocked {
			return fmt.Errorf("market instrument sync lock was not held for %s", strings.TrimSpace(exchangeID))
		}
		return nil
	}, true, nil
}

func marketInstrumentSyncLockKey(exchangeID string) (int64, error) {
	exchangeID = strings.ToLower(strings.TrimSpace(exchangeID))
	if exchangeID == "" {
		return 0, fmt.Errorf("market instrument sync lock exchange is required")
	}
	digest := sha256.Sum256([]byte(marketInstrumentSyncLockKeyPrefix + exchangeID))
	return int64(binary.BigEndian.Uint64(digest[:8])), nil
}
