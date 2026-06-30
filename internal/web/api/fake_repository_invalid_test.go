package api

import (
	"context"
	"strconv"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListDataSyncTaskInvalidIssues(
	_ context.Context,
	id string,
	query data.DataSyncInvalidIssueQuery,
) (data.DataSyncInvalidIssueList, error) {
	query = data.NormalizeDataSyncInvalidIssueQuery(query)
	for _, task := range repository.tasks {
		if task.ID != id {
			continue
		}
		if detail, ok := repository.taskInvalidDetails[id]; ok {
			return paginateDataSyncInvalidIssueList(cloneDataSyncInvalidIssueList(detail), query), nil
		}
		result := data.DataSyncInvalidIssueList{
			TaskID:     task.ID,
			Issues:     []data.CandleIssue{},
			IssueLimit: query.Limit,
			Offset:     query.Offset,
		}
		if task.InvalidSummary != nil && task.InvalidSummary.FirstIssue != nil && task.InvalidSummary.Count > 0 {
			result.Issues = append(result.Issues, *task.InvalidSummary.FirstIssue)
			result.TotalCount = task.InvalidSummary.Count
		}
		return paginateDataSyncInvalidIssueList(result, query), nil
	}
	return data.DataSyncInvalidIssueList{}, data.ErrNotFound
}

func (repository *fakeRepository) RepairDataSyncTaskInvalidIssues(
	_ context.Context,
	id string,
	request data.RepairDataSyncInvalidIssuesRequest,
) (data.DataSyncGapRepairResult, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID != id {
			continue
		}
		if err := repository.requireFakeDataSyncRepairSourceActive(repository.tasks[index]); err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		duration, err := data.IntervalDuration(repository.tasks[index].Interval)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		query := dataSyncInvalidIssueRepairQuery(request)
		detail, err := repository.ListDataSyncTaskInvalidIssues(context.Background(), id, query)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result := data.DataSyncGapRepairResult{
			SourceTaskID: repository.tasks[index].ID,
			CreatedTasks: []data.DataSyncTask{},
			Limited:      detail.Limited,
			TotalCount:   detail.TotalCount,
			RepairLimit:  query.Limit,
		}
		for _, issue := range detail.Issues {
			if issue.OpenTime == nil {
				continue
			}
			from := issue.OpenTime.UTC()
			gap := data.CandleGap{From: from, To: from.Add(duration), MissingCandles: 1}
			if repository.fakeRepairTaskExists(repository.tasks[index], gap) {
				result.SkippedExisting++
				continue
			}
			now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
			repairTask := data.DataSyncTask{
				ID:                 "dst_invalid_repair_" + strconv.Itoa(len(repository.tasks)+1),
				Exchange:           repository.tasks[index].Exchange,
				Symbol:             repository.tasks[index].Symbol,
				Interval:           repository.tasks[index].Interval,
				StartTime:          &gap.From,
				EndTime:            &gap.To,
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
		}
		return result, nil
	}
	return data.DataSyncGapRepairResult{}, data.ErrNotFound
}

func cloneDataSyncInvalidIssueList(value data.DataSyncInvalidIssueList) data.DataSyncInvalidIssueList {
	return data.DataSyncInvalidIssueList{
		TaskID:        value.TaskID,
		Issues:        append([]data.CandleIssue(nil), value.Issues...),
		Limited:       value.Limited,
		TotalCount:    value.TotalCount,
		ReturnedCount: value.ReturnedCount,
		IssueLimit:    value.IssueLimit,
		Offset:        value.Offset,
	}
}

func paginateDataSyncInvalidIssueList(
	value data.DataSyncInvalidIssueList,
	query data.DataSyncInvalidIssueQuery,
) data.DataSyncInvalidIssueList {
	totalCount := value.TotalCount
	filtered := value.Issues
	if hasDataSyncInvalidIssueFilters(query) {
		filtered = make([]data.CandleIssue, 0, len(value.Issues))
		for _, issue := range value.Issues {
			if !matchesDataSyncInvalidIssueQuery(issue, query) {
				continue
			}
			filtered = append(filtered, issue)
		}
		totalCount = len(filtered)
	} else if totalCount == 0 {
		totalCount = len(filtered)
	}
	start := query.Offset
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + query.Limit
	if end > len(filtered) {
		end = len(filtered)
	}
	value.Issues = append([]data.CandleIssue(nil), filtered[start:end]...)
	value.TotalCount = totalCount
	value.ReturnedCount = len(value.Issues)
	value.IssueLimit = query.Limit
	value.Offset = query.Offset
	value.Limited = query.Offset+len(value.Issues) < totalCount
	return value
}

func dataSyncInvalidIssueRepairQuery(request data.RepairDataSyncInvalidIssuesRequest) data.DataSyncInvalidIssueQuery {
	query := data.DataSyncInvalidIssueQuery{
		Limit:  20,
		Code:   request.Code,
		From:   request.From,
		To:     request.To,
		Offset: 0,
	}
	return data.NormalizeDataSyncInvalidIssueQuery(query)
}

func hasDataSyncInvalidIssueFilters(query data.DataSyncInvalidIssueQuery) bool {
	return query.Code != "" || query.From != nil || query.To != nil
}

func matchesDataSyncInvalidIssueQuery(issue data.CandleIssue, query data.DataSyncInvalidIssueQuery) bool {
	if query.Code != "" && issue.Code != query.Code {
		return false
	}
	if issue.OpenTime == nil {
		return query.From == nil && query.To == nil
	}
	if query.From != nil && issue.OpenTime.Before(*query.From) {
		return false
	}
	if query.To != nil && issue.OpenTime.After(*query.To) {
		return false
	}
	return true
}
