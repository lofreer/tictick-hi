package data

import (
	"context"
)

type SyncFetchLockRepository interface {
	TryLockDataSyncExchangeFetch(ctx context.Context, exchange string) (func(context.Context) error, bool, error)
	ReleaseDataSyncTaskAfterSkippedFetch(ctx context.Context, taskID string) error
}

type SyncRepositoryWithFetchLock interface {
	SyncRepository
	SyncFetchLockRepository
}
