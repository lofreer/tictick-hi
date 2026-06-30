package datasync

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/lofreer/tictick-hi/internal/data"
)

func isDataSyncLeaseRace(err error) bool {
	return errors.Is(err, data.ErrNotFound) || errors.Is(err, data.ErrInvalidState)
}

func (runner *Runner) releaseDataSyncTaskOnShutdown(ctx context.Context, task data.DataSyncTask) error {
	err := runner.repository.ReleaseDataSyncTask(ctx, task.ID, runner.config.WorkerID)
	if err == nil {
		return nil
	}
	if isDataSyncLeaseRace(err) {
		slog.Info("data sync task no longer owned before shutdown release", "task_id", task.ID, "error", err)
		return nil
	}
	return fmt.Errorf("release data sync task on shutdown: %w", err)
}

func (runner *Runner) releaseDataSyncTaskAfterExchangeFetchLockSkip(
	ctx context.Context,
	task data.DataSyncTask,
) error {
	err := runner.releaseDataSyncTaskAfterSkippedFetch(ctx, task.ID)
	if err == nil {
		return nil
	}
	if isDataSyncLeaseRace(err) {
		slog.Info("data sync task no longer owned after exchange fetch lock skip", "task_id", task.ID, "error", err)
		return nil
	}
	return fmt.Errorf("release data sync task after exchange fetch lock skip: %w", err)
}

func (runner *Runner) releaseDataSyncTaskAfterExchangeFetchLockError(
	ctx context.Context,
	task data.DataSyncTask,
) error {
	err := runner.releaseDataSyncTaskAfterSkippedFetch(ctx, task.ID)
	if err == nil {
		return nil
	}
	if isDataSyncLeaseRace(err) {
		slog.Info("data sync task no longer owned after exchange fetch lock error", "task_id", task.ID, "error", err)
		return nil
	}
	return fmt.Errorf("release data sync task after exchange fetch lock error: %w", err)
}

func (runner *Runner) releaseDataSyncTaskAfterSkippedFetch(ctx context.Context, taskID string) error {
	if runner.lockRepository == nil {
		return runner.repository.ReleaseDataSyncTask(ctx, taskID, runner.config.WorkerID)
	}
	return runner.lockRepository.ReleaseDataSyncTaskAfterSkippedFetch(ctx, taskID, runner.config.WorkerID)
}
