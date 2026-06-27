package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ListOperatorSessions(
	ctx context.Context,
	operatorID string,
	currentTokenHash string,
	now time.Time,
) ([]data.OperatorSession, error) {
	if _, err := store.pool.Exec(ctx, `
		DELETE FROM operator_sessions
		 WHERE expires_at <= $1`, now); err != nil {
		return nil, fmt.Errorf("delete expired operator sessions: %w", err)
	}

	rows, err := store.pool.Query(ctx, `
		SELECT id, operator_id, token_hash, expires_at, created_at, token_hash = $2 AS current
		  FROM operator_sessions
		 WHERE operator_id = $1
		   AND expires_at > $3
		 ORDER BY current DESC, created_at DESC`,
		operatorID,
		currentTokenHash,
		now,
	)
	if err != nil {
		return nil, fmt.Errorf("list operator sessions: %w", err)
	}
	defer rows.Close()

	sessions, err := pgx.CollectRows(rows, scanOperatorSession)
	if err != nil {
		return nil, fmt.Errorf("collect operator sessions: %w", err)
	}
	return sessions, nil
}

func (store *Store) DeleteOperatorSessionByID(
	ctx context.Context,
	operatorID string,
	sessionID string,
	currentTokenHash string,
) error {
	var tokenHash string
	err := store.pool.QueryRow(ctx, `
		SELECT token_hash
		  FROM operator_sessions
		 WHERE id = $1
		   AND operator_id = $2`,
		sessionID,
		operatorID,
	).Scan(&tokenHash)
	if err == pgx.ErrNoRows {
		return data.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get operator session: %w", err)
	}
	if tokenHash == currentTokenHash {
		return data.ErrInvalidState
	}

	tag, err := store.pool.Exec(ctx, `
		DELETE FROM operator_sessions
		 WHERE id = $1
		   AND operator_id = $2`,
		sessionID,
		operatorID,
	)
	if err != nil {
		return fmt.Errorf("delete operator session by id: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return data.ErrNotFound
	}
	return nil
}

func scanOperatorSession(row pgx.CollectableRow) (data.OperatorSession, error) {
	var session data.OperatorSession
	err := row.Scan(
		&session.ID,
		&session.OperatorID,
		&session.TokenHash,
		&session.ExpiresAt,
		&session.CreatedAt,
		&session.Current,
	)
	return session, err
}
