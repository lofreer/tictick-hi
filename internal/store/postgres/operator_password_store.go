package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ChangeOperatorPassword(
	ctx context.Context,
	operatorID string,
	currentTokenHash string,
	currentPassword string,
	newPassword string,
) (int, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin change operator password: %w", err)
	}
	defer tx.Rollback(ctx)

	var username string
	var passwordHash string
	var enabled bool
	err = tx.QueryRow(ctx, `
		SELECT username, password_hash, enabled
		  FROM operators
		 WHERE id = $1
		 FOR UPDATE`,
		operatorID,
	).Scan(&username, &passwordHash, &enabled)
	if err == pgx.ErrNoRows {
		return 0, data.ErrUnauthorized
	}
	if err != nil {
		return 0, fmt.Errorf("get operator password hash: %w", err)
	}
	if !enabled {
		return 0, data.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(currentPassword)); err != nil {
		return 0, data.ErrUnauthorized
	}
	if err := data.ValidateOperatorPasswordForUsername(username, newPassword); err != nil {
		return 0, err
	}
	if passwordMatchesHash(newPassword, passwordHash) {
		return 0, data.OperatorPasswordReusedError()
	}
	historyHashes, err := recentOperatorPasswordHashes(ctx, tx, operatorID)
	if err != nil {
		return 0, err
	}
	for _, historyHash := range historyHashes {
		if passwordMatchesHash(newPassword, historyHash) {
			return 0, data.OperatorPasswordReusedError()
		}
	}

	newPasswordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("hash operator password: %w", err)
	}
	historyID, err := core.NewPrefixedID("oph")
	if err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE operators
		   SET password_hash = $2,
		       updated_at = now()
		 WHERE id = $1`,
		operatorID,
		string(newPasswordHash),
	); err != nil {
		return 0, fmt.Errorf("change operator password: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO operator_password_history (id, operator_id, password_hash)
		VALUES ($1, $2, $3)`,
		historyID,
		operatorID,
		passwordHash,
	); err != nil {
		return 0, fmt.Errorf("record operator password history: %w", err)
	}
	if err := pruneOperatorPasswordHistory(ctx, tx, operatorID); err != nil {
		return 0, err
	}
	tag, err := tx.Exec(ctx, `
		DELETE FROM operator_sessions
		 WHERE operator_id = $1
		   AND token_hash <> $2`,
		operatorID,
		currentTokenHash,
	)
	if err != nil {
		return 0, fmt.Errorf("delete other operator sessions after password change: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit change operator password: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

func passwordMatchesHash(password string, passwordHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) == nil
}

func recentOperatorPasswordHashes(ctx context.Context, tx pgx.Tx, operatorID string) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT password_hash
		  FROM operator_password_history
		 WHERE operator_id = $1
		 ORDER BY created_at DESC, id DESC
		 LIMIT $2`,
		operatorID,
		data.OperatorPasswordHistoryLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("list operator password history: %w", err)
	}
	defer rows.Close()

	hashes := []string{}
	for rows.Next() {
		var passwordHash string
		if err := rows.Scan(&passwordHash); err != nil {
			return nil, fmt.Errorf("scan operator password history: %w", err)
		}
		hashes = append(hashes, passwordHash)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan operator password history: %w", err)
	}
	return hashes, nil
}

func pruneOperatorPasswordHistory(ctx context.Context, tx pgx.Tx, operatorID string) error {
	if _, err := tx.Exec(ctx, `
		DELETE FROM operator_password_history
		 WHERE operator_id = $1
		   AND id NOT IN (
		     SELECT id
		       FROM operator_password_history
		      WHERE operator_id = $1
		      ORDER BY created_at DESC, id DESC
		      LIMIT $2
		   )`,
		operatorID,
		data.OperatorPasswordHistoryLimit,
	); err != nil {
		return fmt.Errorf("prune operator password history: %w", err)
	}
	return nil
}
