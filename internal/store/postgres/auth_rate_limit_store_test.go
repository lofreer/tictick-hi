package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"
)

func TestLoginRateLimitStoreLocksAndClears(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	keyHash := integrationLoginRateLimitKeyHash(t)
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)

	assertLoginAllowed(t, ctx, store, keyHash, now, true)
	if err := store.RecordLoginFailure(ctx, keyHash, now, 2, time.Minute, time.Hour); err != nil {
		t.Fatal(err)
	}
	assertLoginAllowed(t, ctx, store, keyHash, now.Add(time.Second), true)
	if err := store.RecordLoginFailure(ctx, keyHash, now.Add(2*time.Second), 2, time.Minute, time.Hour); err != nil {
		t.Fatal(err)
	}
	assertLoginAllowed(t, ctx, store, keyHash, now.Add(3*time.Second), false)

	if err := store.ClearLoginRateLimit(ctx, keyHash); err != nil {
		t.Fatal(err)
	}
	assertLoginAllowed(t, ctx, store, keyHash, now.Add(4*time.Second), true)
}

func TestLoginRateLimitStoreResetsExpiredWindow(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	keyHash := integrationLoginRateLimitKeyHash(t)
	now := time.Date(2026, 7, 7, 8, 0, 0, 0, time.UTC)
	old := now.Add(-10 * time.Minute)

	if err := store.RecordLoginFailure(ctx, keyHash, old, 2, 5*time.Minute, time.Hour); err != nil {
		t.Fatal(err)
	}
	assertLoginAllowed(t, ctx, store, keyHash, now, true)
	if err := store.RecordLoginFailure(ctx, keyHash, now.Add(time.Second), 2, 5*time.Minute, time.Hour); err != nil {
		t.Fatal(err)
	}
	assertLoginAllowed(t, ctx, store, keyHash, now.Add(2*time.Second), true)
	if err := store.RecordLoginFailure(ctx, keyHash, now.Add(3*time.Second), 2, 5*time.Minute, time.Hour); err != nil {
		t.Fatal(err)
	}
	assertLoginAllowed(t, ctx, store, keyHash, now.Add(4*time.Second), false)
}

func integrationLoginRateLimitKeyHash(t *testing.T) string {
	t.Helper()
	sum := sha256.Sum256([]byte(integrationID("login_rate_limit")))
	return hex.EncodeToString(sum[:])
}

func assertLoginAllowed(
	t *testing.T,
	ctx context.Context,
	store *Store,
	keyHash string,
	now time.Time,
	expected bool,
) {
	t.Helper()
	allowed, err := store.CheckLoginRateLimit(ctx, keyHash, now, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if allowed != expected {
		t.Fatalf("allowed = %v, want %v", allowed, expected)
	}
}
