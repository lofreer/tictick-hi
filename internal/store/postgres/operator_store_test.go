package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestOperatorStoreRejectsDisablingLastEnabledOperator(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("last_operator"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	states := snapshotOtherOperatorStates(t, ctx, store, operator.ID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		restoreOperatorStates(t, cleanupCtx, store, states)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	if _, err := store.pool.Exec(ctx, `
		UPDATE operators
		   SET enabled = false
		 WHERE id <> $1`,
		operator.ID,
	); err != nil {
		t.Fatal(err)
	}

	_, err = store.SetOperatorEnabled(ctx, operator.ID, false)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorEnabled error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorLastEnabledRequired {
		t.Fatalf("SetOperatorEnabled code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorLastEnabledRequired)
	}

	authenticated, err := store.AuthenticateOperator(ctx, operator.Username, "secret123A")
	if err != nil {
		t.Fatalf("operator was disabled: %v", err)
	}
	if !authenticated.Enabled {
		t.Fatalf("operator enabled = false")
	}
}

func TestOperatorStoreRejectsDisablingLastEnabledAdmin(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	admin, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("last_admin"),
		Password: "secret123A",
		Role:     data.OperatorRoleAdmin,
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("plain_operator"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	states := snapshotOtherOperatorStates(t, ctx, store, admin.ID, operator.ID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		restoreOperatorStates(t, cleanupCtx, store, states)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = ANY($1::text[])`, []string{admin.ID, operator.ID})
	})

	if _, err := store.pool.Exec(ctx, `
		UPDATE operators
		   SET enabled = false
		 WHERE id <> ALL($1::text[])`,
		[]string{admin.ID, operator.ID},
	); err != nil {
		t.Fatal(err)
	}

	_, err = store.SetOperatorEnabled(ctx, admin.ID, false)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorEnabled error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorLastAdminRequired {
		t.Fatalf("SetOperatorEnabled code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorLastAdminRequired)
	}
}

func TestOperatorStoreRejectsDemotingLastEnabledAdmin(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	admin, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("demote_last_admin"),
		Password: "secret123A",
		Role:     data.OperatorRoleAdmin,
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("demote_plain_operator"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	states := snapshotOtherOperatorStates(t, ctx, store, admin.ID, operator.ID)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		restoreOperatorStates(t, cleanupCtx, store, states)
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = ANY($1::text[])`, []string{admin.ID, operator.ID})
	})

	if _, err := store.pool.Exec(ctx, `
		UPDATE operators
		   SET enabled = false
		 WHERE id <> ALL($1::text[])`,
		[]string{admin.ID, operator.ID},
	); err != nil {
		t.Fatal(err)
	}

	_, err = store.SetOperatorRole(ctx, admin.ID, data.OperatorRoleOperator)
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("SetOperatorRole error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorLastAdminRequired {
		t.Fatalf("SetOperatorRole code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorLastAdminRequired)
	}

	authenticated, err := store.AuthenticateOperator(ctx, admin.Username, "secret123A")
	if err != nil {
		t.Fatalf("last admin could not authenticate: %v", err)
	}
	if authenticated.Role != data.OperatorRoleAdmin {
		t.Fatalf("last admin role = %q, want admin", authenticated.Role)
	}
}

func TestOperatorStoreChangesPasswordAndRevokesOtherSessions(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("change_password"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	currentSession := data.OperatorSession{
		ID:         integrationID("os_current"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_current"),
		ExpiresAt:  now.Add(time.Hour),
	}
	otherSession := data.OperatorSession{
		ID:         integrationID("os_other"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_other"),
		ExpiresAt:  now.Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, currentSession); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateOperatorSession(ctx, otherSession); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	revokedCount, err := store.ChangeOperatorPassword(
		ctx,
		operator.ID,
		currentSession.TokenHash,
		"secret123A",
		"secret456B",
	)
	if err != nil {
		t.Fatal(err)
	}
	if revokedCount != 1 {
		t.Fatalf("revoked session count = %d, want 1", revokedCount)
	}
	if _, err := store.AuthenticateOperator(ctx, operator.Username, "secret123A"); !errors.Is(err, data.ErrUnauthorized) {
		t.Fatalf("old password authentication error = %v, want unauthorized", err)
	}
	if _, err := store.AuthenticateOperator(ctx, operator.Username, "secret456B"); err != nil {
		t.Fatalf("new password authentication failed: %v", err)
	}
	sessions, err := store.ListOperatorSessions(ctx, operator.ID, currentSession.TokenHash, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || !sessions[0].Current || sessions[0].ID != currentSession.ID {
		t.Fatalf("unexpected sessions after password change: %#v", sessions)
	}
}

func TestOperatorStoreRejectsRecentPasswordReuse(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("password_reuse"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := data.OperatorSession{
		ID:         integrationID("os_reuse"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_reuse"),
		ExpiresAt:  time.Now().UTC().Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret123A", "secret456B"); err != nil {
		t.Fatal(err)
	}
	_, err = store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret456B", "secret123A")
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("ChangeOperatorPassword reuse error = %v, want invalid state", err)
	}
	if code, ok := data.DomainErrorCode(err); !ok || code != data.ErrorCodeOperatorPasswordReused {
		t.Fatalf("ChangeOperatorPassword reuse code = %q, %t; want %q, true", code, ok, data.ErrorCodeOperatorPasswordReused)
	}
	if _, err := store.AuthenticateOperator(ctx, operator.Username, "secret456B"); err != nil {
		t.Fatalf("current password should remain valid: %v", err)
	}
}

func TestOperatorStorePrunesPasswordHistory(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("password_history"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := data.OperatorSession{
		ID:         integrationID("os_history"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_history"),
		ExpiresAt:  time.Now().UTC().Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	currentPassword := "secret123A"
	for _, nextPassword := range []string{
		"secret201A", "secret202B", "secret203C", "secret204D", "secret205E", "secret206F", "secret207G",
	} {
		if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, currentPassword, nextPassword); err != nil {
			t.Fatalf("change password to %s: %v", nextPassword, err)
		}
		currentPassword = nextPassword
	}

	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		  FROM operator_password_history
		 WHERE operator_id = $1`,
		operator.ID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != data.DefaultOperatorPasswordHistoryLimit {
		t.Fatalf("password history count = %d, want %d", count, data.DefaultOperatorPasswordHistoryLimit)
	}
}

func TestOperatorStoreUsesConfiguredPasswordHistoryLimit(t *testing.T) {
	store := openIntegrationStore(t)
	if err := store.SetOperatorPasswordHistoryLimit(1); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("password_history_limit"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := data.OperatorSession{
		ID:         integrationID("os_history_limit"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_history_limit"),
		ExpiresAt:  time.Now().UTC().Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret123A", "secret456B"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret456B", "secret789C"); err != nil {
		t.Fatal(err)
	}
	_, err = store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret789C", "secret456B")
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("recent password reuse error = %v, want invalid state", err)
	}
	if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret789C", "secret123A"); err != nil {
		t.Fatalf("old password outside configured history should be allowed: %v", err)
	}

	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		  FROM operator_password_history
		 WHERE operator_id = $1`,
		operator.ID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("password history count = %d, want 1", count)
	}
}

func TestOperatorStoreCanDisablePasswordHistory(t *testing.T) {
	store := openIntegrationStore(t)
	if err := store.SetOperatorPasswordHistoryLimit(0); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := testContext(t)
	defer cancel()

	operator, err := store.CreateOperator(ctx, data.CreateOperator{
		Username: integrationID("password_history_disabled"),
		Password: "secret123A",
		Enabled:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	session := data.OperatorSession{
		ID:         integrationID("os_history_disabled"),
		OperatorID: operator.ID,
		TokenHash:  integrationID("token_history_disabled"),
		ExpiresAt:  time.Now().UTC().Add(time.Hour),
	}
	if err := store.CreateOperatorSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := testContext(t)
		defer cleanupCancel()
		_, _ = store.pool.Exec(cleanupCtx, `DELETE FROM operators WHERE id = $1`, operator.ID)
	})

	if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret123A", "secret456B"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret456B", "secret123A"); err != nil {
		t.Fatalf("previous password should be allowed when history is disabled: %v", err)
	}
	_, err = store.ChangeOperatorPassword(ctx, operator.ID, session.TokenHash, "secret123A", "secret123A")
	if !errors.Is(err, data.ErrInvalidState) {
		t.Fatalf("current password reuse error = %v, want invalid state", err)
	}

	var count int
	if err := store.pool.QueryRow(ctx, `
		SELECT count(*)
		  FROM operator_password_history
		 WHERE operator_id = $1`,
		operator.ID,
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("password history count = %d, want 0", count)
	}
}

func TestIntegrationOperatorTrimmedUsernameConstraint(t *testing.T) {
	store := openIntegrationStore(t)
	ctx, cancel := testContext(t)
	defer cancel()

	_, err := store.pool.Exec(ctx, `
		INSERT INTO operators (id, username, password_hash)
		VALUES ($1, '   ', 'hash')`,
		integrationID("op_blank_username"),
	)
	if err == nil {
		t.Fatal("expected operator trimmed username violation")
	}
	assertDatabaseConstraintError(t, err, "operators_trimmed_username_check")
}

type operatorEnabledState struct {
	id      string
	enabled bool
}

func snapshotOtherOperatorStates(
	t *testing.T,
	ctx context.Context,
	store *Store,
	operatorIDs ...string,
) []operatorEnabledState {
	t.Helper()
	protected := map[string]bool{}
	for _, operatorID := range operatorIDs {
		protected[operatorID] = true
	}
	rows, err := store.pool.Query(ctx, `
		SELECT id, enabled
		  FROM operators`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	states := []operatorEnabledState{}
	for rows.Next() {
		var state operatorEnabledState
		if err := rows.Scan(&state.id, &state.enabled); err != nil {
			t.Fatal(err)
		}
		if protected[state.id] {
			continue
		}
		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return states
}

func restoreOperatorStates(
	t *testing.T,
	ctx context.Context,
	store *Store,
	states []operatorEnabledState,
) {
	t.Helper()
	for _, state := range states {
		if _, err := store.pool.Exec(ctx, `
			UPDATE operators
			   SET enabled = $2
			 WHERE id = $1`,
			state.id,
			state.enabled,
		); err != nil {
			t.Fatal(err)
		}
	}
}
