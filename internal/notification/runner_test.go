package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func TestRunnerDeliversLocalNotification(t *testing.T) {
	repository := &fakeNotificationRepository{
		delivery: data.NotificationDelivery{
			ID:             "no_1",
			NotificationID: "nt_1",
			Provider:       "local",
			Target:         "default",
			AttemptCount:   1,
			MaxAttempts:    3,
		},
		claimed: true,
	}
	runner := NewRunner(repository, DemoProviders(), Config{WorkerID: "test"})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.deliveredAt == nil || !repository.deliveredAt.Equal(now) {
		t.Fatalf("deliveredAt = %v", repository.deliveredAt)
	}
	if repository.failedErr != nil {
		t.Fatalf("unexpected failure: %v", repository.failedErr)
	}
}

func TestRunnerSchedulesRetry(t *testing.T) {
	repository := &fakeNotificationRepository{
		delivery: data.NotificationDelivery{
			ID:             "no_1",
			NotificationID: "nt_1",
			Provider:       "local",
			Target:         "fail-target",
			AttemptCount:   1,
			MaxAttempts:    3,
		},
		claimed: true,
	}
	runner := NewRunner(repository, DemoProviders(), Config{WorkerID: "test", RetryDelay: time.Minute})
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.failedErr == nil {
		t.Fatal("expected failure")
	}
	if repository.nextAttemptAt == nil || !repository.nextAttemptAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("nextAttemptAt = %v", repository.nextAttemptAt)
	}
}

func TestRunnerStopsRetryAtMaxAttempts(t *testing.T) {
	repository := &fakeNotificationRepository{
		delivery: data.NotificationDelivery{
			ID:             "no_1",
			NotificationID: "nt_1",
			Provider:       "missing",
			Target:         "default",
			AttemptCount:   3,
			MaxAttempts:    3,
		},
		claimed: true,
	}
	runner := NewRunner(repository, DemoProviders(), Config{WorkerID: "test", RetryDelay: time.Minute})

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.failedErr == nil {
		t.Fatal("expected failure")
	}
	if repository.nextAttemptAt != nil {
		t.Fatalf("nextAttemptAt = %v, want nil", repository.nextAttemptAt)
	}
}

func TestRunnerDrainsAvailableDeliveriesBeforeSleeping(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &fakeNotificationRepository{
		deliveries: []data.NotificationDelivery{
			{ID: "no_1", NotificationID: "nt_1", Provider: "local", Target: "default", AttemptCount: 1, MaxAttempts: 3},
			{ID: "no_2", NotificationID: "nt_2", Provider: "local", Target: "default", AttemptCount: 1, MaxAttempts: 3},
		},
		cancelOnEmpty: cancel,
	}
	runner := NewRunner(repository, DemoProviders(), Config{WorkerID: "test", PollInterval: time.Hour})

	err := runner.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Run error = %v, want context canceled", err)
	}
	if repository.deliveredCount != 2 {
		t.Fatalf("deliveredCount = %d, want 2", repository.deliveredCount)
	}
}

type fakeNotificationRepository struct {
	delivery       data.NotificationDelivery
	deliveries     []data.NotificationDelivery
	claimed        bool
	cancelOnEmpty  func()
	deliveredAt    *time.Time
	deliveredCount int
	failedErr      error
	nextAttemptAt  *time.Time
}

func (repository *fakeNotificationRepository) ClaimNotificationDelivery(
	context.Context,
	string,
	time.Duration,
) (data.NotificationDelivery, bool, error) {
	if len(repository.deliveries) > 0 {
		delivery := repository.deliveries[0]
		repository.deliveries = repository.deliveries[1:]
		return delivery, true, nil
	}
	if !repository.claimed {
		if repository.cancelOnEmpty != nil {
			repository.cancelOnEmpty()
		}
		return data.NotificationDelivery{}, false, nil
	}
	repository.claimed = false
	return repository.delivery, true, nil
}

func (repository *fakeNotificationRepository) MarkNotificationDelivered(
	_ context.Context,
	_ string,
	deliveredAt time.Time,
) error {
	repository.deliveredAt = &deliveredAt
	repository.deliveredCount++
	return nil
}

func (repository *fakeNotificationRepository) MarkNotificationFailed(
	_ context.Context,
	_ string,
	err error,
	nextAttemptAt *time.Time,
) error {
	if err == nil {
		return errors.New("expected error")
	}
	repository.failedErr = err
	repository.nextAttemptAt = nextAttemptAt
	return nil
}
