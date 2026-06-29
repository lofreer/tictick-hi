package api

import (
	"context"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) SetSyncEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	if enabled {
		if err := repository.requireActiveTaskMarket(id); err != nil {
			return data.DataSyncTask{}, err
		}
	}
	return repository.updateTask(ctx, id, func(task *data.DataSyncTask) {
		delete(repository.catalogPausedTasks, task.ID)
		task.SyncEnabled = enabled
		if enabled {
			task.LastError = ""
		}
		task.Status = data.TaskStatusPending
		if !enabled {
			task.Status = data.TaskStatusPaused
		}
		refreshFakeDataSyncHealth(task)
	})
}

func (repository *fakeRepository) SetRealtimeEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	if enabled {
		if err := repository.requireActiveTaskMarket(id); err != nil {
			return data.DataSyncTask{}, err
		}
	}
	return repository.updateTask(ctx, id, func(task *data.DataSyncTask) {
		delete(repository.catalogPausedTasks, task.ID)
		task.RealtimeEnabled = enabled
		if enabled {
			task.LastError = ""
		}
		task.Status = data.TaskStatusRunning
		if !enabled {
			task.Status = data.TaskStatusPaused
		}
		refreshFakeDataSyncHealth(task)
	})
}

func (repository *fakeRepository) updateTask(
	_ context.Context,
	id string,
	update func(task *data.DataSyncTask),
) (data.DataSyncTask, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID == id {
			if !dataSyncTaskCommandAllowed(repository.tasks[index].Status) {
				return data.DataSyncTask{}, data.DataSyncCommandInvalidStateError()
			}
			update(&repository.tasks[index])
			repository.tasks[index] = normalizeFakeDataSyncTask(repository.tasks[index])
			return repository.tasks[index], nil
		}
	}
	return data.DataSyncTask{}, data.ErrNotFound
}

func (repository *fakeRepository) requireActiveTaskMarket(id string) error {
	for _, task := range repository.tasks {
		if task.ID != id {
			continue
		}
		if normalizeFakeDataSyncTask(task).MarketStatus != data.DataSyncMarketStatusActive {
			return data.MarketInstrumentNotActiveError()
		}
		return nil
	}
	return data.ErrNotFound
}

func normalizeFakeDataSyncTask(task data.DataSyncTask) data.DataSyncTask {
	if task.MarketStatus == "" {
		task.MarketStatus = data.DataSyncMarketStatusActive
	}
	if task.MarketStatusDetail == "" {
		if task.MarketStatus == data.DataSyncMarketStatusMissing {
			task.MarketStatusDetail = "missing"
		} else {
			task.MarketStatusDetail = string(task.MarketStatus)
		}
	}
	return task
}
