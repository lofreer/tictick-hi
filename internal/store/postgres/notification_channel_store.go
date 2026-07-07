package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

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

func (store *Store) UpdateNotificationChannel(
	ctx context.Context,
	id string,
	channel data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	row := store.pool.QueryRow(ctx, `
		UPDATE notification_channels
		   SET name = $2,
		       provider = $3,
		       target = $4,
		       enabled = $5,
		       updated_at = now()
		 WHERE id = $1
		RETURNING id, name, provider, target, enabled, created_at, updated_at`,
		id,
		channel.Name,
		channel.Provider,
		channel.Target,
		channel.Enabled,
	)
	updated, err := scanNotificationChannelRow(row)
	if err == pgx.ErrNoRows {
		return data.NotificationChannel{}, data.ErrNotFound
	}
	if err != nil {
		return data.NotificationChannel{}, fmt.Errorf("update notification channel: %w", err)
	}
	return updated, nil
}

func (store *Store) DeleteNotificationChannel(ctx context.Context, id string) (data.NotificationChannel, error) {
	row := store.pool.QueryRow(ctx, `
		DELETE FROM notification_channels
		 WHERE id = $1
		RETURNING id, name, provider, target, enabled, created_at, updated_at`,
		id,
	)
	channel, err := scanNotificationChannelRow(row)
	if err == pgx.ErrNoRows {
		return data.NotificationChannel{}, data.ErrNotFound
	}
	if err != nil {
		return data.NotificationChannel{}, fmt.Errorf("delete notification channel: %w", err)
	}
	return channel, nil
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
