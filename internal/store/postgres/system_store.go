package postgres

import (
	"context"
	"errors"
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

func (store *Store) SetNotificationChannelEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.NotificationChannel, error) {
	row := store.pool.QueryRow(ctx, `
		UPDATE notification_channels
		   SET enabled = $2,
		       updated_at = now()
		 WHERE id = $1
		RETURNING id, name, provider, target, enabled, created_at, updated_at`,
		id,
		enabled,
	)
	channel, err := scanNotificationChannelRow(row)
	if err == pgx.ErrNoRows {
		return data.NotificationChannel{}, data.ErrNotFound
	}
	if err != nil {
		return data.NotificationChannel{}, fmt.Errorf("set notification channel enabled: %w", err)
	}
	return channel, nil
}

func (store *Store) ListExchangeAccounts(ctx context.Context) ([]data.ExchangeAccount, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, exchange, alias, enabled,
		       CASE
		         WHEN encrypted_api_key LIKE 'v1:aesgcm:%'
		          AND encrypted_api_secret LIKE 'v1:aesgcm:%' THEN 'encrypted'
		         ELSE 'legacy'
		       END,
		       created_at, updated_at
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
	if store.secretBox == nil {
		return data.ExchangeAccount{}, errors.New("ENCRYPTION_KEY is required to store exchange account credentials")
	}
	id, err := core.NewPrefixedID("ea")
	if err != nil {
		return data.ExchangeAccount{}, err
	}
	encryptedAPIKey, err := store.secretBox.Seal(account.APIKey)
	if err != nil {
		return data.ExchangeAccount{}, fmt.Errorf("encrypt api key: %w", err)
	}
	encryptedAPISecret, err := store.secretBox.Seal(account.APISecret)
	if err != nil {
		return data.ExchangeAccount{}, fmt.Errorf("encrypt api secret: %w", err)
	}
	row := store.pool.QueryRow(ctx, `
		INSERT INTO exchange_accounts (
			id, exchange, alias, encrypted_api_key, encrypted_api_secret, enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, exchange, alias, enabled, 'encrypted', created_at, updated_at`,
		id,
		account.Exchange,
		account.Alias,
		encryptedAPIKey,
		encryptedAPISecret,
		account.Enabled,
	)

	created, err := scanExchangeAccountRow(row)
	if err != nil {
		return data.ExchangeAccount{}, fmt.Errorf("create exchange account: %w", err)
	}
	return created, nil
}

func (store *Store) GetExchangeAccount(ctx context.Context, exchange string, accountID string) (data.ExchangeAccount, error) {
	row := store.pool.QueryRow(ctx, `
		SELECT id, exchange, alias, enabled,
		       CASE
		         WHEN encrypted_api_key LIKE 'v1:aesgcm:%'
		          AND encrypted_api_secret LIKE 'v1:aesgcm:%' THEN 'encrypted'
		         ELSE 'legacy'
		       END,
		       created_at, updated_at
		  FROM exchange_accounts
		 WHERE exchange = $1
		   AND (id = $2 OR alias = $2)
		 ORDER BY created_at DESC
		 LIMIT 1`,
		exchange,
		accountID,
	)
	account, err := scanExchangeAccountRow(row)
	if err == pgx.ErrNoRows {
		return data.ExchangeAccount{}, data.ErrNotFound
	}
	if err != nil {
		return data.ExchangeAccount{}, fmt.Errorf("get exchange account: %w", err)
	}
	return account, nil
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
	if err := data.ValidateOperatorPasswordForUsername(operator.Username, operator.Password); err != nil {
		return data.Operator{}, err
	}
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

func (store *Store) SetOperatorEnabled(ctx context.Context, id string, enabled bool) (data.Operator, error) {
	if enabled {
		return setOperatorEnabled(ctx, store.pool, id, enabled)
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.Operator{}, fmt.Errorf("begin set operator enabled: %w", err)
	}
	defer tx.Rollback(ctx)

	enabledIDs, err := lockedEnabledOperatorIDs(ctx, tx)
	if err != nil {
		return data.Operator{}, err
	}
	if containsString(enabledIDs, id) && len(enabledIDs) <= 1 {
		return data.Operator{}, data.OperatorLastEnabledError()
	}

	operator, err := setOperatorEnabled(ctx, tx, id, enabled)
	if err != nil {
		return data.Operator{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return data.Operator{}, fmt.Errorf("commit set operator enabled: %w", err)
	}
	return operator, nil
}

type operatorEnabledQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func setOperatorEnabled(
	ctx context.Context,
	queryer operatorEnabledQueryer,
	id string,
	enabled bool,
) (data.Operator, error) {
	row := queryer.QueryRow(ctx, `
		UPDATE operators
		   SET enabled = $2,
		       updated_at = now()
		 WHERE id = $1
		RETURNING id, username, enabled, created_at, updated_at`,
		id,
		enabled,
	)
	operator, err := scanOperatorRow(row)
	if err == pgx.ErrNoRows {
		return data.Operator{}, data.ErrNotFound
	}
	if err != nil {
		return data.Operator{}, fmt.Errorf("set operator enabled: %w", err)
	}
	return operator, nil
}

func lockedEnabledOperatorIDs(ctx context.Context, tx pgx.Tx) ([]string, error) {
	rows, err := tx.Query(ctx, `
		SELECT id
		  FROM operators
		 WHERE enabled = true
		 ORDER BY id
		 FOR UPDATE`)
	if err != nil {
		return nil, fmt.Errorf("lock enabled operators: %w", err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan enabled operator id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scan enabled operator ids: %w", err)
	}
	return ids, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
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
		INSERT INTO operator_sessions (id, token_hash, operator_id, expires_at, remote_addr, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (token_hash)
		DO UPDATE SET
			id = EXCLUDED.id,
			operator_id = EXCLUDED.operator_id,
			expires_at = EXCLUDED.expires_at,
			remote_addr = EXCLUDED.remote_addr,
			user_agent = EXCLUDED.user_agent`,
		session.ID,
		session.TokenHash,
		session.OperatorID,
		session.ExpiresAt,
		session.RemoteAddr,
		session.UserAgent,
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
		&account.CredentialStatus,
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
