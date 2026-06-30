package postgres

import (
	"strings"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/exchange"
)

func TestIntegrationDataSyncExchangeBackoffDoesNotBlockOtherExchanges(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	clearIntegrationExchangeBackoff(t, ctx, store, "binance")
	clearIntegrationExchangeBackoff(t, ctx, store, "okx")

	binanceID := integrationID("dst_binance_backoff")
	binanceSiblingID := integrationID("dst_binance_backoff_sibling")
	okxID := integrationID("dst_okx_backoff")
	binanceSymbol := integrationSymbol("BX")
	binanceSiblingSymbol := integrationSymbol("BS")
	okxSymbol := strings.TrimSuffix(integrationSymbol("OX"), "USDT") + "-USDT"
	insertIntegrationSyncTask(t, ctx, store, binanceID, binanceSymbol, data.TaskStatusRunning, true, true, "exchange-backoff-worker")
	insertIntegrationSyncTask(t, ctx, store, binanceSiblingID, binanceSiblingSymbol, data.TaskStatusPending, true, false, "")
	insertIntegrationSyncTaskForExchange(t, ctx, store, "okx", okxID, okxSymbol, data.TaskStatusPending, true, false, "")
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_tasks WHERE id IN ($1, $2, $3)`, binanceID, binanceSiblingID, okxID)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM market_instruments WHERE symbol IN ($1, $2, $3)`, binanceSymbol, binanceSiblingSymbol, okxSymbol)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM data_sync_exchange_backoffs WHERE exchange IN ('binance', 'okx')`)
	})

	if err := store.RecordDataSyncRetry(
		ctx,
		binanceID,
		"exchange-backoff-worker",
		exchange.NewTemporaryError("binance klines temporary unavailable: api.binance.com: EOF", nil),
		ptrTime(time.Now().UTC().Add(time.Hour)),
	); err != nil {
		t.Fatal(err)
	}

	claimed, ok, err := store.ClaimDataSyncTask(ctx, "okx-claim-worker", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("okx task should still be claimable while binance exchange backoff is active")
	}
	if claimed.ID != okxID || claimed.Exchange != "okx" {
		t.Fatalf("claimed task = %#v, want okx task %s", claimed, okxID)
	}

	binanceRow := readIntegrationSyncTask(t, ctx, store, binanceID)
	if binanceRow.lockedBy.Valid || binanceRow.lockedUntil.Valid || binanceRow.heartbeatAt.Valid {
		t.Fatalf("binance retry task should remain released during backoff: %#v", binanceRow)
	}
	binanceSiblingRow := readIntegrationSyncTask(t, ctx, store, binanceSiblingID)
	if binanceSiblingRow.lockedBy.Valid || binanceSiblingRow.lockedUntil.Valid || binanceSiblingRow.heartbeatAt.Valid {
		t.Fatalf("binance sibling task should not be claimed during binance backoff: %#v", binanceSiblingRow)
	}
	okxRow := readIntegrationSyncTask(t, ctx, store, okxID)
	if okxRow.status != data.TaskStatusRunning ||
		!okxRow.lockedBy.Valid ||
		okxRow.lockedBy.String != "okx-claim-worker" {
		t.Fatalf("okx task should be claimed despite binance backoff: %#v", okxRow)
	}
}
