package postgres

import (
	"errors"
	"testing"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestNotificationChannelStoreSetEnabled(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	channel, err := store.CreateNotificationChannel(ctx, data.CreateNotificationChannel{
		Name:     integrationID("channel"),
		Provider: "local",
		Target:   "default",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM notification_channels WHERE id = $1`, channel.ID)
	})

	disabled, err := store.SetNotificationChannelEnabled(ctx, channel.ID, false)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Enabled {
		t.Fatalf("channel enabled = true after disable")
	}
	if disabled.UpdatedAt.Before(channel.UpdatedAt) {
		t.Fatalf("updatedAt moved backwards: before=%s after=%s", channel.UpdatedAt, disabled.UpdatedAt)
	}

	enabled, err := store.SetNotificationChannelEnabled(ctx, channel.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	if !enabled.Enabled {
		t.Fatalf("channel enabled = false after enable")
	}

	_, err = store.SetNotificationChannelEnabled(ctx, "nc_missing", false)
	if !errors.Is(err, data.ErrNotFound) {
		t.Fatalf("missing channel error = %v, want not found", err)
	}
}
