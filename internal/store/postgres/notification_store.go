package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (store *Store) ListNotifications(ctx context.Context) ([]data.Notification, error) {
	rows, err := store.pool.Query(ctx, `
		SELECT id, task_id, COALESCE(intent_id, ''), channel, provider, target,
		       title, body, status, COALESCE(error, ''), attempt_count,
		       max_attempts, next_attempt_at, last_attempt_at, created_at, sent_at
		  FROM notifications
		 ORDER BY created_at DESC
		 LIMIT 200`)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, scanNotification)
}

func (store *Store) RetryNotification(ctx context.Context, id string) (data.Notification, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.Notification{}, fmt.Errorf("begin retry notification: %w", err)
	}
	defer tx.Rollback(ctx)

	notification, err := notificationByID(ctx, tx, id)
	if err != nil {
		return data.Notification{}, err
	}
	route, err := notificationRoute(ctx, tx, notification.Channel)
	if err != nil {
		return data.Notification{}, err
	}
	if !route.Enabled {
		const disabledError = "notification channel is disabled"
		if err := markRetriedNotificationDisabled(ctx, tx, id, route, disabledError); err != nil {
			return data.Notification{}, err
		}
		notification, err := notificationByID(ctx, tx, id)
		if err != nil {
			return data.Notification{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return data.Notification{}, fmt.Errorf("commit retry disabled notification: %w", err)
		}
		return notification, nil
	}

	commandTag, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE notification_outbox
		   SET status = 'pending',
		       provider = $2,
		       target = $3,
		       next_attempt_at = now(),
		       %s,
		       last_error = NULL,
		       updated_at = now()
		 WHERE notification_id = $1`, clearLeaseAssignments(notificationOutboxLease)),
		id,
		route.Provider,
		route.Target,
	)
	if err != nil {
		return data.Notification{}, fmt.Errorf("retry notification outbox: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return data.Notification{}, data.ErrNotFound
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notifications
		   SET status = 'pending',
		       provider = $2,
		       target = $3,
		       error = NULL,
		       next_attempt_at = now(),
		       sent_at = NULL,
		       updated_at = now()
		 WHERE id = $1`,
		id,
		route.Provider,
		route.Target,
	); err != nil {
		return data.Notification{}, fmt.Errorf("retry notification: %w", err)
	}

	notification, err = notificationByID(ctx, tx, id)
	if err != nil {
		return data.Notification{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return data.Notification{}, fmt.Errorf("commit retry notification: %w", err)
	}
	return notification, nil
}

func markRetriedNotificationDisabled(
	ctx context.Context,
	tx pgx.Tx,
	id string,
	route resolvedNotificationRoute,
	message string,
) error {
	commandTag, err := tx.Exec(ctx, fmt.Sprintf(`
		UPDATE notification_outbox
		   SET status = 'failed',
		       provider = $2,
		       target = $3,
		       next_attempt_at = NULL,
		       %s,
		       last_error = $4,
		       updated_at = now()
		 WHERE notification_id = $1`, clearLeaseAssignments(notificationOutboxLease)),
		id,
		route.Provider,
		route.Target,
		message,
	)
	if err != nil {
		return fmt.Errorf("mark disabled notification outbox: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return data.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE notifications
		   SET status = 'failed',
		       provider = $2,
		       target = $3,
		       error = $4,
		       next_attempt_at = NULL,
		       updated_at = now()
		 WHERE id = $1`,
		id,
		route.Provider,
		route.Target,
		message,
	); err != nil {
		return fmt.Errorf("mark disabled notification: %w", err)
	}
	return nil
}

func (store *Store) ClaimNotificationDelivery(
	ctx context.Context,
	workerID string,
	leaseTTL time.Duration,
) (data.NotificationDelivery, bool, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.NotificationDelivery{}, false, fmt.Errorf("begin claim notification delivery: %w", err)
	}
	defer tx.Rollback(ctx)

	var id string
	err = tx.QueryRow(ctx, fmt.Sprintf(`
		SELECT id
		  FROM notification_outbox
		 WHERE (
		       (status IN ('pending', 'retry_scheduled') AND next_attempt_at <= now())
		       OR status = 'running'
		   )
		   AND %s
		 ORDER BY next_attempt_at ASC, created_at ASC
		 LIMIT 1
		 FOR UPDATE SKIP LOCKED`,
		claimableLeasePredicate(),
	)).Scan(&id)
	if err == pgx.ErrNoRows {
		return data.NotificationDelivery{}, false, nil
	}
	if err != nil {
		return data.NotificationDelivery{}, false, fmt.Errorf("select notification outbox: %w", err)
	}

	row := tx.QueryRow(ctx, fmt.Sprintf(`
		UPDATE notification_outbox
		   SET status = 'running',
		       %s
		 WHERE id = $1
		RETURNING id, notification_id, task_id, COALESCE(intent_id, ''), channel,
		          provider, target, title, body, status, attempt_count, max_attempts,
		          next_attempt_at, last_attempt_at, COALESCE(last_error, ''),
		          created_at, updated_at`,
		claimLeaseAssignments(
			notificationOutboxLease,
			"$2",
			"$3",
			"last_attempt_at = now()",
		),
	),
		id, workerID, intervalLiteral(leaseTTL),
	)
	delivery, err := scanNotificationDelivery(row)
	if err != nil {
		return data.NotificationDelivery{}, false, fmt.Errorf("update notification outbox claim: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notifications
		   SET status = 'running',
		       attempt_count = $2,
		       last_attempt_at = $3,
		       updated_at = now()
		 WHERE id = $1`,
		delivery.NotificationID,
		delivery.AttemptCount,
		delivery.LastAttemptAt,
	); err != nil {
		return data.NotificationDelivery{}, false, fmt.Errorf("mark notification running: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.NotificationDelivery{}, false, fmt.Errorf("commit claim notification delivery: %w", err)
	}
	return delivery, true, nil
}

func (store *Store) MarkNotificationDelivered(ctx context.Context, deliveryID string, deliveredAt time.Time) error {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mark notification delivered: %w", err)
	}
	defer tx.Rollback(ctx)

	var notificationID string
	var attemptCount int
	err = tx.QueryRow(ctx, fmt.Sprintf(`
		UPDATE notification_outbox
		   SET status = 'delivered',
		       delivered_at = $2,
		       last_error = NULL,
		       %s,
		       updated_at = now()
		 WHERE id = $1
		RETURNING notification_id, attempt_count`, clearLeaseAssignments(notificationOutboxLease)),
		deliveryID,
		deliveredAt,
	).Scan(&notificationID, &attemptCount)
	if err == pgx.ErrNoRows {
		return data.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("mark notification outbox delivered: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notifications
		   SET status = 'sent',
		       error = NULL,
		       attempt_count = $2,
		       sent_at = $3,
		       updated_at = now()
		 WHERE id = $1`,
		notificationID,
		attemptCount,
		deliveredAt,
	); err != nil {
		return fmt.Errorf("mark notification delivered: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit notification delivered: %w", err)
	}
	return nil
}

func (store *Store) MarkNotificationFailed(
	ctx context.Context,
	deliveryID string,
	taskErr error,
	nextAttemptAt *time.Time,
) error {
	status := "failed"
	var nextAttempt any
	if nextAttemptAt != nil {
		status = "retry_scheduled"
		nextAttempt = *nextAttemptAt
	}
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin mark notification failed: %w", err)
	}
	defer tx.Rollback(ctx)

	var notificationID string
	var attemptCount int
	err = tx.QueryRow(ctx, fmt.Sprintf(`
		UPDATE notification_outbox
		   SET status = $2,
		       next_attempt_at = $3,
		       last_error = $4,
		       %s,
		       updated_at = now()
		 WHERE id = $1
		RETURNING notification_id, attempt_count`, clearLeaseAssignments(notificationOutboxLease)),
		deliveryID,
		status,
		nextAttempt,
		normalizeTaskError(taskErr),
	).Scan(&notificationID, &attemptCount)
	if err == pgx.ErrNoRows {
		return data.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("mark notification outbox failed: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		UPDATE notifications
		   SET status = $2,
		       error = $3,
		       attempt_count = $4,
		       next_attempt_at = $5,
		       updated_at = now()
		 WHERE id = $1`,
		notificationID,
		status,
		normalizeTaskError(taskErr),
		attemptCount,
		nextAttempt,
	); err != nil {
		return fmt.Errorf("mark notification failed: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit notification failed: %w", err)
	}
	return nil
}

func (store *Store) ReleaseNotificationDelivery(ctx context.Context, deliveryID string) error {
	if err := releaseLease(ctx, store.pool, notificationOutboxLease, deliveryID); err != nil {
		return fmt.Errorf("release notification delivery: %w", err)
	}
	return nil
}

func notificationByID(ctx context.Context, tx pgx.Tx, id string) (data.Notification, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, task_id, COALESCE(intent_id, ''), channel, provider, target,
		       title, body, status, COALESCE(error, ''), attempt_count,
		       max_attempts, next_attempt_at, last_attempt_at, created_at, sent_at
		  FROM notifications
		 WHERE id = $1`, id)
	notification, err := scanNotificationRow(row)
	if err == pgx.ErrNoRows {
		return data.Notification{}, data.ErrNotFound
	}
	if err != nil {
		return data.Notification{}, fmt.Errorf("get notification: %w", err)
	}
	return notification, nil
}

func scanNotificationDelivery(row rowScanner) (data.NotificationDelivery, error) {
	var delivery data.NotificationDelivery
	err := row.Scan(
		&delivery.ID,
		&delivery.NotificationID,
		&delivery.TaskID,
		&delivery.IntentID,
		&delivery.Channel,
		&delivery.Provider,
		&delivery.Target,
		&delivery.Title,
		&delivery.Body,
		&delivery.Status,
		&delivery.AttemptCount,
		&delivery.MaxAttempts,
		&delivery.NextAttemptAt,
		&delivery.LastAttemptAt,
		&delivery.LastError,
		&delivery.CreatedAt,
		&delivery.UpdatedAt,
	)
	return delivery, err
}
