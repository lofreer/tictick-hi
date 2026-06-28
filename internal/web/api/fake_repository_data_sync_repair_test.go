package api

import (
	"context"
	"strconv"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) RepairDataSyncTaskGap(
	_ context.Context,
	id string,
	request data.RepairDataSyncTaskGapRequest,
) (data.DataSyncGapRepairResult, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID != id {
			continue
		}
		result := data.DataSyncGapRepairResult{
			SourceTaskID: repository.tasks[index].ID,
			CreatedTasks: []data.DataSyncTask{},
			TotalCount:   1,
			RepairLimit:  1,
		}
		gap := data.CandleGap{From: request.From, To: request.To}
		if repository.fakeRepairTaskExists(repository.tasks[index], gap) {
			result.SkippedExisting = 1
			return result, nil
		}
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		repairTask := data.DataSyncTask{
			ID:                 "dst_repair_" + strconv.Itoa(len(repository.tasks)+1),
			Exchange:           repository.tasks[index].Exchange,
			Symbol:             repository.tasks[index].Symbol,
			Interval:           repository.tasks[index].Interval,
			StartTime:          &request.From,
			EndTime:            &request.To,
			RepairSourceTaskID: repository.tasks[index].ID,
			SyncEnabled:        true,
			RealtimeEnabled:    false,
			Status:             data.TaskStatusPending,
			DataHealth:         data.DataSyncHealthSyncing,
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		repository.tasks = append(repository.tasks, repairTask)
		result.CreatedTasks = append(result.CreatedTasks, repairTask)
		return result, nil
	}
	return data.DataSyncGapRepairResult{}, data.ErrNotFound
}
