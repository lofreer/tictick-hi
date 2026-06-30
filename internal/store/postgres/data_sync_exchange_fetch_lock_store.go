package postgres

import "context"

const dataSyncExchangeFetchLockKeyPrefix = "tictick-hi:data-sync-exchange-fetch:"

func (store *Store) TryLockDataSyncExchangeFetch(
	ctx context.Context,
	exchangeID string,
) (func(context.Context) error, bool, error) {
	return store.tryExchangeAdvisoryLock(ctx, dataSyncExchangeFetchLockKeyPrefix, exchangeID, "data sync exchange fetch")
}
