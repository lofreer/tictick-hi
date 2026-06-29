package api

import (
	"context"

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
	if totalCount == 0 {
		totalCount = len(value.Issues)
	}
	start := query.Offset
	if start > len(value.Issues) {
		start = len(value.Issues)
	}
	end := start + query.Limit
	if end > len(value.Issues) {
		end = len(value.Issues)
	}
	value.Issues = append([]data.CandleIssue(nil), value.Issues[start:end]...)
	value.TotalCount = totalCount
	value.ReturnedCount = len(value.Issues)
	value.IssueLimit = query.Limit
	value.Offset = query.Offset
	value.Limited = query.Offset+len(value.Issues) < totalCount
	return value
}
