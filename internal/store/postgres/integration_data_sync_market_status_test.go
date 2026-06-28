package postgres

import (
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestIntegrationClaimDataSyncTaskSkipsInactiveMarketInstrument(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	inactiveID := integrationID("dst")
	activeID := integrationID("dst")
	inactiveSymbol := integrationSymbol("CSI")
	activeSymbol := integrationSymbol("CSA")
	insertIntegrationSyncTask(t, ctx, store, inactiveID, inactiveSymbol, data.TaskStatusPending, true, false, "")
	insertIntegrationSyncTask(t, ctx, store, activeID, activeSymbol, data.TaskStatusPending, true, false, "")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id IN ($1, $2)`, inactiveID, activeID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2)`, inactiveSymbol, activeSymbol)
	})
	if _, err := store.pool.Exec(ctx, `UPDATE market_instruments SET status = 'inactive' WHERE exchange = 'binance' AND symbol = $1`, inactiveSymbol); err != nil {
		t.Fatal(err)
	}

	claimed, ok, err := store.ClaimDataSyncTask(ctx, "inactive-skip-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected active task to be claimed")
	}
	if claimed.ID != activeID || claimed.MarketStatus != data.DataSyncMarketStatusActive {
		t.Fatalf("claimed task = %#v, want active task %s", claimed, activeID)
	}
}
