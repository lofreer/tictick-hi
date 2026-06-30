package postgres

import (
	"strings"
	"testing"
	"time"
)

func TestIntegrationSystemHealthReportsStaleMarketInstrumentCatalog(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	exchangeID := "binance"
	staleSuccessAt := time.Now().UTC().Add(-25 * time.Hour).Truncate(time.Second)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `
			UPDATE market_instrument_sync_statuses
			   SET last_attempt_at = now(),
			       last_success_at = now(),
			       last_error = '',
			       updated_at = now()
			 WHERE exchange = $1`,
			exchangeID,
		)
	})

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO market_instrument_sync_statuses (
			exchange, last_attempt_at, last_success_at, last_error, updated_at
		)
		VALUES ($1, $2, $2, '', $2)
		ON CONFLICT (exchange) DO UPDATE
		   SET last_attempt_at = EXCLUDED.last_attempt_at,
		       last_success_at = EXCLUDED.last_success_at,
		       last_error = '',
		       updated_at = EXCLUDED.updated_at`,
		exchangeID,
		staleSuccessAt,
	); err != nil {
		t.Fatal(err)
	}

	health, err := store.SystemHealth(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "degraded" {
		t.Fatalf("system health status = %q, want degraded", health.Status)
	}
	catalogHealth := findIntegrationServiceHealth(health, "market-instrument-catalog")
	if catalogHealth.Status != "warning" ||
		!strings.Contains(catalogHealth.Detail, "binance") ||
		!strings.Contains(catalogHealth.Detail, "stale_since=") {
		t.Fatalf("unexpected stale catalog health: %#v", catalogHealth)
	}
}
