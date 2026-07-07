package postgres

import "testing"

func TestIntegrationExchangeAccountTrimmedRequiredTextConstraint(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO exchange_accounts (
			id, exchange, alias, encrypted_api_key, encrypted_api_secret
		)
		VALUES ($1, '   ', 'main', 'v1:aesgcm:key', 'v1:aesgcm:secret')`,
		integrationID("ex_blank_exchange"),
	)
	if err == nil {
		t.Fatal("expected exchange account trimmed required text violation")
	}
	assertDatabaseConstraintError(t, err, "exchange_accounts_trimmed_required_text_check")
}
