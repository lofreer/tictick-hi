package api

import (
	"context"

	"github.com/lofreer/tictick-hi/internal/data"
)

func (repository *fakeRepository) ListDataSyncTaskInvalidIssues(
	_ context.Context,
	id string,
) (data.DataSyncInvalidIssueList, error) {
	for _, task := range repository.tasks {
		if task.ID != id {
			continue
		}
		if detail, ok := repository.taskInvalidDetails[id]; ok {
			return cloneDataSyncInvalidIssueList(detail), nil
		}
		result := data.DataSyncInvalidIssueList{
			TaskID:     task.ID,
			Issues:     []data.CandleIssue{},
			IssueLimit: 50,
		}
		if task.InvalidSummary != nil && task.InvalidSummary.FirstIssue != nil && task.InvalidSummary.Count > 0 {
			result.Issues = append(result.Issues, *task.InvalidSummary.FirstIssue)
			result.TotalCount = task.InvalidSummary.Count
			result.ReturnedCount = len(result.Issues)
			result.Limited = task.InvalidSummary.Count > result.ReturnedCount
		}
		return result, nil
	}
	return data.DataSyncInvalidIssueList{}, data.ErrNotFound
}

func cloneDataSyncInvalidIssueList(value data.DataSyncInvalidIssueList) data.DataSyncInvalidIssueList {
	return data.DataSyncInvalidIssueList{
		TaskID:        value.TaskID,
		Issues:        append([]data.CandleIssue(nil), value.Issues...),
		Limited:       value.Limited,
		TotalCount:    value.TotalCount,
		ReturnedCount: value.ReturnedCount,
		IssueLimit:    value.IssueLimit,
	}
}
