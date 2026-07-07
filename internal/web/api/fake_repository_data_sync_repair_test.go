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
		if err := repository.requireFakeDataSyncRepairSourceActive(repository.tasks[index]); err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result := data.DataSyncGapRepairResult{
			SourceTaskID: repository.tasks[index].ID,
			CreatedTasks: []data.DataSyncTask{},
			TotalCount:   1,
			RepairLimit:  1,
		}
		gap := data.CandleGap{From: request.From, To: request.To}
		if !repository.fakeRepairableTaskGap(id, gap) {
			return data.DataSyncGapRepairResult{}, data.ErrNotFound
		}
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
			RequestID:          request.RequestID,
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

func (repository *fakeRepository) RepairDataSyncTaskGaps(
	_ context.Context,
	id string,
	request data.RepairDataSyncTaskGapsRequest,
) (data.DataSyncGapRepairResult, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID != id {
			continue
		}
		if err := repository.requireFakeDataSyncRepairSourceActive(repository.tasks[index]); err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result := data.DataSyncGapRepairResult{
			SourceTaskID: repository.tasks[index].ID,
			CreatedTasks: []data.DataSyncTask{},
			RepairLimit:  20,
		}
		summary := repository.tasks[index].GapSummary
		if summary == nil || summary.FirstGap == nil || summary.Count <= 0 {
			return result, nil
		}
		result.TotalCount = summary.Count
		result.Limited = summary.Count > result.RepairLimit
		if repository.fakeRepairTaskExists(repository.tasks[index], *summary.FirstGap) {
			result.SkippedExisting = 1
			return result, nil
		}
		now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		repairTask := data.DataSyncTask{
			ID:                 "dst_repair_" + strconv.Itoa(len(repository.tasks)+1),
			Exchange:           repository.tasks[index].Exchange,
			Symbol:             repository.tasks[index].Symbol,
			Interval:           repository.tasks[index].Interval,
			StartTime:          &summary.FirstGap.From,
			EndTime:            &summary.FirstGap.To,
			RepairSourceTaskID: repository.tasks[index].ID,
			RequestID:          request.RequestID,
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

func (repository *fakeRepository) requireFakeDataSyncRepairSourceActive(task data.DataSyncTask) error {
	if normalizeFakeDataSyncTask(task).MarketStatus != data.DataSyncMarketStatusActive {
		return data.MarketInstrumentNotActiveError()
	}
	return nil
}

func (repository *fakeRepository) fakeRepairableTaskGap(id string, gap data.CandleGap) bool {
	if detail, ok := repository.taskGapDetails[id]; ok {
		for _, candidate := range detail.Gaps {
			if candidate.From.Equal(gap.From) && candidate.To.Equal(gap.To) {
				return true
			}
		}
		return false
	}
	for _, task := range repository.tasks {
		if task.ID != id || task.GapSummary == nil || task.GapSummary.FirstGap == nil || task.GapSummary.Count <= 0 {
			continue
		}
		return task.GapSummary.FirstGap.From.Equal(gap.From) && task.GapSummary.FirstGap.To.Equal(gap.To)
	}
	return false
}
