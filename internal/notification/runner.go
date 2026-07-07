package notification

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/workerlease"
	"github.com/lofreer/tictick-hi/internal/workerlog"
)

type Runner struct {
	repository               data.NotificationRepository
	providers                ProviderRegistry
	config                   Config
	now                      func() time.Time
	sleep                    func(context.Context, time.Duration) error
	lastProviderDeliveryTime time.Time
}

type Config struct {
	WorkerID            string
	LeaseTTL            time.Duration
	PollInterval        time.Duration
	RetryDelay          time.Duration
	MaxRetryDelay       time.Duration
	ProviderMinInterval time.Duration
}

func NewRunner(repository data.NotificationRepository, providers ProviderRegistry, config Config) *Runner {
	if config.WorkerID == "" {
		config.WorkerID = "notify-worker"
	}
	if config.LeaseTTL <= 0 {
		config.LeaseTTL = 30 * time.Second
	}
	if config.PollInterval <= 0 {
		config.PollInterval = 10 * time.Second
	}
	if config.RetryDelay <= 0 {
		config.RetryDelay = 30 * time.Second
	}
	if config.MaxRetryDelay <= 0 {
		config.MaxRetryDelay = 5 * time.Minute
	}
	if config.ProviderMinInterval < 0 {
		config.ProviderMinInterval = 0
	}
	return &Runner{
		repository: repository,
		providers:  providers,
		config:     config,
		now:        func() time.Time { return time.Now().UTC() },
		sleep:      sleepContext,
	}
}

func (runner *Runner) Run(ctx context.Context) error {
	ticker := time.NewTicker(runner.config.PollInterval)
	defer ticker.Stop()

	for {
		for {
			processed, err := runner.runOne(ctx)
			if err != nil {
				if workerlease.IsShutdown(ctx, err) {
					return nil
				}
				return err
			}
			if !processed {
				break
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func (runner *Runner) RunOnce(ctx context.Context) error {
	_, err := runner.runOne(ctx)
	return err
}

func (runner *Runner) runOne(ctx context.Context) (bool, error) {
	delivery, ok, err := runner.repository.ClaimNotificationDelivery(
		ctx,
		runner.config.WorkerID,
		runner.config.LeaseTTL,
	)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	if err := runner.deliver(ctx, delivery); err != nil {
		if workerlease.IsShutdown(ctx, err) {
			releaseCtx, cancel := workerlease.ReleaseContext(ctx)
			defer cancel()
			if releaseErr := runner.repository.ReleaseNotificationDelivery(releaseCtx, delivery.ID); releaseErr != nil {
				return true, fmt.Errorf("release notification delivery on shutdown: %w", releaseErr)
			}
			return true, nil
		}
		slog.Error(
			"notification delivery failed",
			workerlog.TaskTraceAttrs(
				delivery.TaskID,
				delivery.RequestID,
				delivery.TraceParent,
				"delivery_id",
				delivery.ID,
				"error",
				err,
			)...,
		)
		nextAttemptAt := runner.nextAttemptAt(delivery)
		if markErr := runner.repository.MarkNotificationFailed(ctx, delivery.ID, err, nextAttemptAt); markErr != nil {
			return true, fmt.Errorf("mark notification failed: %w", markErr)
		}
		return true, nil
	}
	if err := runner.repository.MarkNotificationDelivered(ctx, delivery.ID, runner.now()); err != nil {
		return true, fmt.Errorf("mark notification delivered: %w", err)
	}
	return true, nil
}

func (runner *Runner) deliver(ctx context.Context, delivery data.NotificationDelivery) error {
	provider, err := runner.providers.Provider(delivery.Provider)
	if err != nil {
		return err
	}
	if err := runner.throttleProvider(ctx); err != nil {
		return err
	}
	err = provider.Deliver(ctx, delivery)
	runner.lastProviderDeliveryTime = runner.now()
	return err
}

func (runner *Runner) throttleProvider(ctx context.Context) error {
	if runner.config.ProviderMinInterval <= 0 || runner.lastProviderDeliveryTime.IsZero() {
		return nil
	}
	wait := runner.lastProviderDeliveryTime.Add(runner.config.ProviderMinInterval).Sub(runner.now())
	if wait <= 0 {
		return nil
	}
	return runner.sleep(ctx, wait)
}

func (runner *Runner) nextAttemptAt(delivery data.NotificationDelivery) *time.Time {
	if delivery.AttemptCount >= delivery.MaxAttempts {
		return nil
	}
	delay := runner.config.RetryDelay * time.Duration(delivery.AttemptCount)
	if delay <= 0 {
		delay = runner.config.RetryDelay
	}
	if delay > runner.config.MaxRetryDelay {
		delay = runner.config.MaxRetryDelay
	}
	next := runner.now().Add(delay)
	return &next
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
