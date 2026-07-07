package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) SetOperatorRole(
	ctx context.Context,
	id string,
	role string,
) (data.OperatorRoleUpdateResult, error) {
	role = data.NormalizeOperatorRole(role)
	if err := data.ValidateOperatorRole(role); err != nil {
		return data.OperatorRoleUpdateResult{}, err
	}
	if role == data.OperatorRoleAdmin {
		return setOperatorRole(ctx, store.pool, id, role)
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.OperatorRoleUpdateResult{}, fmt.Errorf("begin set operator role: %w", err)
	}
	defer tx.Rollback(ctx)

	enabledOperators, err := lockedEnabledOperators(ctx, tx)
	if err != nil {
		return data.OperatorRoleUpdateResult{}, err
	}
	if target, ok := findLockedOperator(enabledOperators, id); ok &&
		target.Role == data.OperatorRoleAdmin &&
		enabledAdminOperatorCount(enabledOperators) <= 1 {
		return data.OperatorRoleUpdateResult{}, data.OperatorLastAdminError()
	}

	result, err := setOperatorRole(ctx, tx, id, role)
	if err != nil {
		return data.OperatorRoleUpdateResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return data.OperatorRoleUpdateResult{}, fmt.Errorf("commit set operator role: %w", err)
	}
	return result, nil
}

func setOperatorRole(
	ctx context.Context,
	queryer operatorEnabledQueryer,
	id string,
	role string,
) (data.OperatorRoleUpdateResult, error) {
	var result data.OperatorRoleUpdateResult
	row := queryer.QueryRow(ctx, `
		WITH previous AS (
			SELECT id, role AS previous_role
			  FROM operators
			 WHERE id = $1
		), updated AS (
			UPDATE operators
			   SET role = $2,
			       updated_at = now()
			 WHERE id = $1
			RETURNING id, username, role, enabled, created_at, updated_at
		)
		SELECT updated.id, updated.username, updated.role, updated.enabled,
		       updated.created_at, updated.updated_at, previous.previous_role
		  FROM updated
		  JOIN previous ON previous.id = updated.id`,
		id,
		role,
	)
	err := row.Scan(
		&result.Operator.ID,
		&result.Operator.Username,
		&result.Operator.Role,
		&result.Operator.Enabled,
		&result.Operator.CreatedAt,
		&result.Operator.UpdatedAt,
		&result.PreviousRole,
	)
	if err == pgx.ErrNoRows {
		return data.OperatorRoleUpdateResult{}, data.ErrNotFound
	}
	if err != nil {
		return data.OperatorRoleUpdateResult{}, fmt.Errorf("set operator role: %w", err)
	}
	return result, nil
}
