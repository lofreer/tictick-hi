package postgres

import "testing"

func TestIntegrationMarketInstrumentSyncLockIsExclusivePerExchange(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	unlockBinance, locked, err := store.TryLockMarketInstrumentSync(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("first binance lock was not acquired")
	}
	defer func() {
		if unlockBinance != nil {
			if err := unlockBinance(ctx); err != nil {
				t.Fatal(err)
			}
		}
	}()

	unlockSecondBinance, locked, err := store.TryLockMarketInstrumentSync(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if locked {
		if unlockSecondBinance != nil {
			_ = unlockSecondBinance(ctx)
		}
		t.Fatal("second binance lock was acquired while first lock was held")
	}
	if unlockSecondBinance != nil {
		t.Fatal("second binance unlock function should be nil when lock is not acquired")
	}

	unlockOKX, locked, err := store.TryLockMarketInstrumentSync(ctx, "okx")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("okx lock was not acquired while binance lock was held")
	}
	if err := unlockOKX(ctx); err != nil {
		t.Fatal(err)
	}

	if err := unlockBinance(ctx); err != nil {
		t.Fatal(err)
	}
	unlockBinance = nil

	unlockAfterRelease, locked, err := store.TryLockMarketInstrumentSync(ctx, "binance")
	if err != nil {
		t.Fatal(err)
	}
	if !locked {
		t.Fatal("binance lock was not reacquired after release")
	}
	if err := unlockAfterRelease(ctx); err != nil {
		t.Fatal(err)
	}
}
