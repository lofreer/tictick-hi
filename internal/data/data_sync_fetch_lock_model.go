package data

import (
	"context"
	"time"
)

type SyncFetchLockRepository interface {
	TryLockDataSyncExchangeFetch(ctx context.Context, exchange string) (func(context.Context) error, bool, error)
	ReleaseDataSyncTaskAfterSkippedFetch(ctx context.Context, taskID string) error
	RecordDataSyncExchangeFetchLockSkipped(ctx context.Context, exchange string, skippedAt time.Time) error
}

type SyncRepositoryWithFetchLock interface {
	SyncRepository
	SyncFetchLockRepository
}
