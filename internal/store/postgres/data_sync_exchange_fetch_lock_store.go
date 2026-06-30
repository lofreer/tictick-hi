package postgres

import (
	"context"
	"fmt"
	"time"
)

const dataSyncExchangeFetchLockKeyPrefix = "tictick-hi:data-sync-exchange-fetch:"

func (store *Store) TryLockDataSyncExchangeFetch(
	ctx context.Context,
	exchangeID string,
) (func(context.Context) error, bool, error) {
	return store.tryExchangeAdvisoryLock(ctx, dataSyncExchangeFetchLockKeyPrefix, exchangeID, "data sync exchange fetch")
}

func (store *Store) RecordDataSyncExchangeFetchLockSkipped(
	ctx context.Context,
	exchange string,
	skippedAt time.Time,
) error {
	if skippedAt.IsZero() {
		skippedAt = time.Now().UTC()
	}
	if _, err := store.pool.Exec(ctx, `
		INSERT INTO data_sync_exchange_fetch_lock_skips (exchange, skip_count, last_skipped_at, updated_at)
		VALUES ($1, 1, $2, now())
		ON CONFLICT (exchange) DO UPDATE
		   SET skip_count = data_sync_exchange_fetch_lock_skips.skip_count + 1,
		       last_skipped_at = GREATEST(
		         data_sync_exchange_fetch_lock_skips.last_skipped_at,
		         EXCLUDED.last_skipped_at
		       ),
		       updated_at = now()`,
		exchange,
		skippedAt.UTC(),
	); err != nil {
		return fmt.Errorf("record data sync exchange fetch lock skip: %w", err)
	}
	return nil
}
