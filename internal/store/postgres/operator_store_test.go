package postgres

import (
	"context"
	"errors"
	"testing"

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
	operatorID string,
) []operatorEnabledState {
	t.Helper()
	rows, err := store.pool.Query(ctx, `
		SELECT id, enabled
		  FROM operators
		 WHERE id <> $1`,
		operatorID,
	)
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
