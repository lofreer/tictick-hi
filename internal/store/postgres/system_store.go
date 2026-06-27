package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/lofreer/tictick-hi/internal/core"
	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ListNotificationChannels(ctx context.Context) ([]data.NotificationChannel, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, name, provider, target, enabled, created_at, updated_at
		  FROM notification_channels
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanNotificationChannel)
}

func (store *Store) CreateNotificationChannel(
	ctx context.Context,
	channel data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	id, err := core.NewPrefixedID("nc")
	if err != nil {
		return data.NotificationChannel{}, err
	}
	row := store.pool.QueryRow(ctx, `
		INSERT INTO notification_channels (id, name, provider, target, enabled)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, provider, target, enabled, created_at, updated_at`,
		id,
		channel.Name,
		channel.Provider,
		channel.Target,
		channel.Enabled,
	)

	created, err := scanNotificationChannelRow(row)
	if err != nil {
		return data.NotificationChannel{}, fmt.Errorf("create notification channel: %w", err)
	}
	return created, nil
}

func (store *Store) ListExchangeAccounts(ctx context.Context) ([]data.ExchangeAccount, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, exchange, alias, enabled, created_at, updated_at
		  FROM exchange_accounts
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list exchange accounts: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanExchangeAccount)
}

func (store *Store) CreateExchangeAccount(
	ctx context.Context,
	account data.CreateExchangeAccount,
) (data.ExchangeAccount, error) {
	id, err := core.NewPrefixedID("ea")
	if err != nil {
		return data.ExchangeAccount{}, err
	}
	row := store.pool.QueryRow(ctx, `
		INSERT INTO exchange_accounts (
			id, exchange, alias, encrypted_api_key, encrypted_api_secret, enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, exchange, alias, enabled, created_at, updated_at`,
		id,
		account.Exchange,
		account.Alias,
		secretDigest(account.APIKey),
		secretDigest(account.APISecret),
		account.Enabled,
	)

	created, err := scanExchangeAccountRow(row)
	if err != nil {
		return data.ExchangeAccount{}, fmt.Errorf("create exchange account: %w", err)
	}
	return created, nil
}

func (store *Store) ListOperators(ctx context.Context) ([]data.Operator, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, username, enabled, created_at, updated_at
		  FROM operators
		 ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list operators: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanOperator)
}

func (store *Store) CreateOperator(ctx context.Context, operator data.CreateOperator) (data.Operator, error) {
	id, err := core.NewPrefixedID("op")
	if err != nil {
		return data.Operator{}, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(operator.Password), bcrypt.DefaultCost)
	if err != nil {
		return data.Operator{}, fmt.Errorf("hash operator password: %w", err)
	}

	row := store.pool.QueryRow(ctx, `
		INSERT INTO operators (id, username, password_hash, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id, username, enabled, created_at, updated_at`,
		id,
		operator.Username,
		string(passwordHash),
		operator.Enabled,
	)

	created, err := scanOperatorRow(row)
	if err != nil {
		return data.Operator{}, fmt.Errorf("create operator: %w", err)
	}
	return created, nil
}

func (store *Store) EnsureOperator(
	ctx context.Context,
	operator data.CreateOperator,
) (data.Operator, bool, error) {
	row := store.pool.QueryRow(ctx, `
		SELECT id, username, enabled, created_at, updated_at
		  FROM operators
		 WHERE username = $1`, operator.Username)

	existing, err := scanOperatorRow(row)
	if err == nil {
		return existing, false, nil
	}
	if err != pgx.ErrNoRows {
		return data.Operator{}, false, fmt.Errorf("get operator: %w", err)
	}

	created, err := store.CreateOperator(ctx, operator)
	if err != nil {
		return data.Operator{}, false, err
	}
	return created, true, nil
}

func (store *Store) AuthenticateOperator(
	ctx context.Context,
	username string,
	password string,
) (data.Operator, error) {
	var operator data.Operator
	var passwordHash string
	err := store.pool.QueryRow(ctx, `
		SELECT id, username, password_hash, enabled, created_at, updated_at
		  FROM operators
		 WHERE username = $1`, username).Scan(
		&operator.ID,
		&operator.Username,
		&passwordHash,
		&operator.Enabled,
		&operator.CreatedAt,
		&operator.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return data.Operator{}, data.ErrUnauthorized
	}
	if err != nil {
		return data.Operator{}, fmt.Errorf("authenticate operator: %w", err)
	}
	if !operator.Enabled {
		return data.Operator{}, data.ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)); err != nil {
		return data.Operator{}, data.ErrUnauthorized
	}
	return operator, nil
}

func (store *Store) CreateOperatorSession(ctx context.Context, session data.OperatorSession) error {
	if _, err := store.pool.Exec(ctx, `
		DELETE FROM operator_sessions
		 WHERE expires_at <= now()`); err != nil {
		return fmt.Errorf("delete expired operator sessions: %w", err)
	}

	if _, err := store.pool.Exec(ctx, `
		INSERT INTO operator_sessions (token_hash, operator_id, expires_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (token_hash)
		DO UPDATE SET operator_id = EXCLUDED.operator_id, expires_at = EXCLUDED.expires_at`,
		session.TokenHash,
		session.OperatorID,
		session.ExpiresAt,
	); err != nil {
		return fmt.Errorf("create operator session: %w", err)
	}
	return nil
}

func (store *Store) GetOperatorBySession(
	ctx context.Context,
	tokenHash string,
	now time.Time,
) (data.Operator, error) {
	row := store.pool.QueryRow(ctx, `
		SELECT operators.id, operators.username, operators.enabled, operators.created_at, operators.updated_at
		  FROM operator_sessions
		  JOIN operators ON operators.id = operator_sessions.operator_id
		 WHERE operator_sessions.token_hash = $1
		   AND operator_sessions.expires_at > $2
		   AND operators.enabled = true`,
		tokenHash,
		now,
	)

	operator, err := scanOperatorRow(row)
	if err == pgx.ErrNoRows {
		return data.Operator{}, data.ErrUnauthorized
	}
	if err != nil {
		return data.Operator{}, fmt.Errorf("get operator by session: %w", err)
	}
	return operator, nil
}

func (store *Store) DeleteOperatorSession(ctx context.Context, tokenHash string) error {
	if _, err := store.pool.Exec(ctx, `
		DELETE FROM operator_sessions
		 WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete operator session: %w", err)
	}
	return nil
}

func (store *Store) SystemHealth(ctx context.Context) (data.SystemHealth, error) {
	checkedAt := time.Now().UTC()
	if err := store.pool.Ping(ctx); err != nil {
		return data.SystemHealth{
			Status:    "degraded",
			Database:  "failed",
			CheckedAt: checkedAt,
			Services: []data.ServiceHealth{
				{Name: "postgres", Status: "failed", Detail: err.Error()},
				{Name: "api", Status: "ok"},
			},
		}, nil
	}
	return data.SystemHealth{
		Status:    "ok",
		Database:  "ok",
		CheckedAt: checkedAt,
		Services: []data.ServiceHealth{
			{Name: "postgres", Status: "ok"},
			{Name: "api", Status: "ok"},
			{Name: "sync-worker", Status: "external"},
			{Name: "backtest-worker", Status: "external"},
			{Name: "trading-worker", Status: "external"},
		},
	}, nil
}

func scanNotificationChannel(row pgx.CollectableRow) (data.NotificationChannel, error) {
	return scanNotificationChannelRow(row)
}

func scanNotificationChannelRow(row rowScanner) (data.NotificationChannel, error) {
	var channel data.NotificationChannel
	err := row.Scan(
		&channel.ID,
		&channel.Name,
		&channel.Provider,
		&channel.Target,
		&channel.Enabled,
		&channel.CreatedAt,
		&channel.UpdatedAt,
	)
	return channel, err
}

func scanExchangeAccount(row pgx.CollectableRow) (data.ExchangeAccount, error) {
	return scanExchangeAccountRow(row)
}

func scanExchangeAccountRow(row rowScanner) (data.ExchangeAccount, error) {
	var account data.ExchangeAccount
	err := row.Scan(
		&account.ID,
		&account.Exchange,
		&account.Alias,
		&account.Enabled,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	return account, err
}

func scanOperator(row pgx.CollectableRow) (data.Operator, error) {
	return scanOperatorRow(row)
}

func scanOperatorRow(row rowScanner) (data.Operator, error) {
	var operator data.Operator
	err := row.Scan(
		&operator.ID,
		&operator.Username,
		&operator.Enabled,
		&operator.CreatedAt,
		&operator.UpdatedAt,
	)
	return operator, err
}

func secretDigest(value string) string {
	digest := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(digest[:])
}
