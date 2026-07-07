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

func TestRunnerRecordsProviderDeliveryDurationOnSuccess(t *testing.T) {
	repository := &fakeNotificationRepository{
		delivery: data.NotificationDelivery{
			ID:             "no_1",
			NotificationID: "nt_1",
			Provider:       "timing",
			Target:         "default",
			AttemptCount:   1,
			MaxAttempts:    3,
		},
		claimed: true,
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runner := NewRunner(
		repository,
		ProviderRegistry{providers: map[string]Provider{"timing": timingProvider{now: &now, duration: 375 * time.Millisecond}}},
		Config{WorkerID: "test"},
	)
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.deliveredDuration != 375*time.Millisecond {
		t.Fatalf("delivered duration = %s", repository.deliveredDuration)
	}
	if repository.deliveredAt == nil || !repository.deliveredAt.Equal(now) {
		t.Fatalf("deliveredAt = %v, now = %v", repository.deliveredAt, now)
	}
}

func TestRunnerRecordsProviderDeliveryDurationOnFailure(t *testing.T) {
	repository := &fakeNotificationRepository{
		delivery: data.NotificationDelivery{
			ID:             "no_1",
			NotificationID: "nt_1",
			Provider:       "timing",
			Target:         "default",
			AttemptCount:   1,
			MaxAttempts:    3,
		},
		claimed: true,
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runner := NewRunner(
		repository,
		ProviderRegistry{providers: map[string]Provider{
			"timing": timingProvider{now: &now, duration: 125 * time.Millisecond, err: errors.New("provider failed")},
		}},
		Config{WorkerID: "test", RetryDelay: time.Minute},
	)
	runner.now = func() time.Time { return now }

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.failedDuration != 125*time.Millisecond {
		t.Fatalf("failed duration = %s", repository.failedDuration)
	}
	if repository.nextAttemptAt == nil || !repository.nextAttemptAt.Equal(now.Add(time.Minute)) {
		t.Fatalf("nextAttemptAt = %v, now = %v", repository.nextAttemptAt, now)
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
	if err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if repository.deliveredCount != 2 {
		t.Fatalf("deliveredCount = %d, want 2", repository.deliveredCount)
	}
}

func TestRunnerAppliesProviderMinIntervalBetweenDeliveries(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &fakeNotificationRepository{
		deliveries: []data.NotificationDelivery{
			{ID: "no_1", NotificationID: "nt_1", Provider: "counting", Target: "default", AttemptCount: 1, MaxAttempts: 3},
			{ID: "no_2", NotificationID: "nt_2", Provider: "counting", Target: "default", AttemptCount: 1, MaxAttempts: 3},
		},
		cancelOnEmpty: cancel,
	}
	var delivered int
	runner := NewRunner(
		repository,
		ProviderRegistry{providers: map[string]Provider{"counting": countingProvider{count: &delivered}}},
		Config{WorkerID: "test", PollInterval: time.Hour, ProviderMinInterval: 250 * time.Millisecond},
	)
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	runner.now = func() time.Time { return now }
	var sleeps []time.Duration
	runner.sleep = func(_ context.Context, duration time.Duration) error {
		sleeps = append(sleeps, duration)
		now = now.Add(duration)
		return nil
	}

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run error = %v", err)
	}
	if delivered != 2 || repository.deliveredCount != 2 {
		t.Fatalf("delivered provider/repository counts = %d/%d", delivered, repository.deliveredCount)
	}
	if len(sleeps) != 1 || sleeps[0] != 250*time.Millisecond {
		t.Fatalf("provider sleeps = %#v", sleeps)
	}
}

func TestRunnerReleasesDeliveryOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	repository := &fakeNotificationRepository{
		delivery: data.NotificationDelivery{
			ID:             "no_1",
			NotificationID: "nt_1",
			Provider:       "canceling",
			Target:         "default",
			AttemptCount:   1,
			MaxAttempts:    3,
		},
		claimed: true,
	}
	runner := NewRunner(
		repository,
		ProviderRegistry{providers: map[string]Provider{"canceling": cancelingProvider{cancel: cancel}}},
		Config{WorkerID: "test"},
	)

	if err := runner.RunOnce(ctx); err != nil {
		t.Fatal(err)
	}
	if !repository.released {
		t.Fatal("expected delivery lease to be released on shutdown")
	}
	if repository.failedErr != nil {
		t.Fatalf("shutdown should not mark notification failed: %v", repository.failedErr)
	}
	if repository.deliveredAt != nil {
		t.Fatalf("shutdown should not mark notification delivered: %v", repository.deliveredAt)
	}
}

type fakeNotificationRepository struct {
	delivery          data.NotificationDelivery
	deliveries        []data.NotificationDelivery
	claimed           bool
	cancelOnEmpty     func()
	deliveredAt       *time.Time
	deliveredDuration time.Duration
	deliveredCount    int
	failedErr         error
	failedDuration    time.Duration
	nextAttemptAt     *time.Time
	released          bool
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
	deliveryDuration time.Duration,
) error {
	repository.deliveredAt = &deliveredAt
	repository.deliveredDuration = deliveryDuration
	repository.deliveredCount++
	return nil
}

func (repository *fakeNotificationRepository) MarkNotificationFailed(
	_ context.Context,
	_ string,
	err error,
	nextAttemptAt *time.Time,
	deliveryDuration time.Duration,
) error {
	if err == nil {
		return errors.New("expected error")
	}
	repository.failedErr = err
	repository.failedDuration = deliveryDuration
	repository.nextAttemptAt = nextAttemptAt
	return nil
}

func (repository *fakeNotificationRepository) ReleaseNotificationDelivery(context.Context, string) error {
	repository.released = true
	return nil
}

type cancelingProvider struct {
	cancel func()
}

func (provider cancelingProvider) Deliver(ctx context.Context, _ data.NotificationDelivery) error {
	provider.cancel()
	<-ctx.Done()
	return ctx.Err()
}

type countingProvider struct {
	count *int
}

func (provider countingProvider) Deliver(context.Context, data.NotificationDelivery) error {
	(*provider.count)++
	return nil
}

type timingProvider struct {
	now      *time.Time
	duration time.Duration
	err      error
}

func (provider timingProvider) Deliver(context.Context, data.NotificationDelivery) error {
	*provider.now = (*provider.now).Add(provider.duration)
	return provider.err
}
