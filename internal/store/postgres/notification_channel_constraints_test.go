package postgres

import "testing"

func TestIntegrationNotificationChannelTrimmedRequiredTextConstraint(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO notification_channels (id, name, provider, target)
		VALUES ($1, '   ', 'local', 'default')`,
		integrationID("nc_blank_name"),
	)
	if err == nil {
		t.Fatal("expected notification channel trimmed required text violation")
	}
	assertDatabaseConstraintError(t, err, "notification_channels_trimmed_required_text_check")
}
