package postgres

import "context"

const marketInstrumentSyncLockKeyPrefix = "tictick-hi:market-instrument-sync:"

func (store *Store) TryLockMarketInstrumentSync(
	ctx context.Context,
	exchangeID string,
) (func(context.Context) error, bool, error) {
	return store.tryExchangeAdvisoryLock(ctx, marketInstrumentSyncLockKeyPrefix, exchangeID, "market instrument sync")
}
