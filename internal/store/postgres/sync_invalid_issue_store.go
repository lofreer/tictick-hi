package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const (
	maxDataSyncInvalidIssues           = data.MaxDataSyncInvalidIssueLimit
	maxDataSyncInvalidIssueRepairTasks = 20
)

func (store *Store) ListDataSyncTaskInvalidIssues(
	ctx context.Context,
	id string,
	query data.DataSyncInvalidIssueQuery,
) (data.DataSyncInvalidIssueList, error) {
	query = data.NormalizeDataSyncInvalidIssueQuery(query)
	task, err := getDataSyncTask(ctx, store.pool, id)
	if err != nil {
		return data.DataSyncInvalidIssueList{}, err
	}

	issues, totalCount, err := listDataSyncInvalidIssues(ctx, store.pool, id, query)
	if err != nil {
		return data.DataSyncInvalidIssueList{}, err
	}
	return data.DataSyncInvalidIssueList{
		TaskID:        task.ID,
		Issues:        issues,
		Limited:       query.Offset+len(issues) < totalCount,
		TotalCount:    totalCount,
		ReturnedCount: len(issues),
		IssueLimit:    query.Limit,
		Offset:        query.Offset,
	}, nil
}

func (store *Store) RepairDataSyncTaskInvalidIssues(
	ctx context.Context,
	id string,
	request data.RepairDataSyncInvalidIssuesRequest,
) (data.DataSyncGapRepairResult, error) {
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("begin repair data sync invalid issues: %w", err)
	}
	defer tx.Rollback(ctx)

	source, err := lockDataSyncTask(ctx, tx, id)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if err := data.ValidateDataSyncTaskWindow(source.Interval, nil, nil); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	if err := ensureDataSyncRepairSourceMarketActive(ctx, tx, source); err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	duration, err := data.IntervalDuration(source.Interval)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}

	query := dataSyncInvalidIssueRepairQuery(request)
	issues, totalCount, err := listDataSyncInvalidIssues(ctx, tx, id, query)
	if err != nil {
		return data.DataSyncGapRepairResult{}, err
	}
	result := data.DataSyncGapRepairResult{
		SourceTaskID: source.ID,
		CreatedTasks: []data.DataSyncTask{},
		Limited:      totalCount > len(issues),
		TotalCount:   totalCount,
		RepairLimit:  maxDataSyncInvalidIssueRepairTasks,
	}
	for _, issue := range issues {
		if issue.OpenTime == nil || !data.IsRepairableCandleIssueCode(issue.Code) {
			continue
		}
		from := issue.OpenTime.UTC()
		window := dataSyncGapRepairWindow{
			from:           from,
			to:             from.Add(duration),
			missingCandles: 1,
		}
		exists, err := dataSyncRepairTaskExists(ctx, tx, source, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		if exists {
			result.SkippedExisting++
			continue
		}
		task, err := insertDataSyncRepairTask(ctx, tx, source, window)
		if err != nil {
			return data.DataSyncGapRepairResult{}, err
		}
		result.CreatedTasks = append(result.CreatedTasks, task)
	}

	if err := tx.Commit(ctx); err != nil {
		return data.DataSyncGapRepairResult{}, fmt.Errorf("commit repair data sync invalid issues: %w", err)
	}
	return result, nil
}

func listDataSyncInvalidIssues(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	id string,
	query data.DataSyncInvalidIssueQuery,
) ([]data.CandleIssue, int, error) {
	from, to := dataSyncInvalidIssueTimeBounds(query)
	const invalidFilterSQL = `
		 WHERE invalid_code IS NOT NULL
		   AND ($2 = '' OR invalid_code = $2)
		   AND ($3::timestamptz IS NULL OR open_time >= $3)
		   AND ($4::timestamptz IS NULL OR open_time <= $4)`

	var totalCount int
	if err := queryer.QueryRow(ctx, fmt.Sprintf(`
		WITH %s
		SELECT COUNT(*)
		  FROM ordered_candles
		%s`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
		invalidFilterSQL,
	), id, query.Code, from, to).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count data sync invalid issues: %w", err)
	}
	if totalCount == 0 || query.Offset >= totalCount {
		return []data.CandleIssue{}, totalCount, nil
	}

	rows, err := queryer.Query(ctx, fmt.Sprintf(`
		WITH %s
		SELECT open_time, invalid_code, invalid_message
		  FROM ordered_candles
		%s
		 ORDER BY open_time
		 LIMIT $5 OFFSET $6`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
		invalidFilterSQL,
	), id, query.Code, from, to, query.Limit, query.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list data sync invalid issues: %w", err)
	}
	defer rows.Close()

	issues := make([]data.CandleIssue, 0)
	for rows.Next() {
		var openTime time.Time
		var issue data.CandleIssue
		if err := rows.Scan(&openTime, &issue.Code, &issue.Message); err != nil {
			return nil, 0, fmt.Errorf("scan data sync invalid issue: %w", err)
		}
		issue.OpenTime = &openTime
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate data sync invalid issues: %w", err)
	}

	return issues, totalCount, nil
}

func dataSyncInvalidIssueRepairQuery(request data.RepairDataSyncInvalidIssuesRequest) data.DataSyncInvalidIssueQuery {
	query := data.DataSyncInvalidIssueQuery{
		Limit: maxDataSyncInvalidIssueRepairTasks,
		Code:  request.Code,
	}
	if request.From != nil {
		value := request.From.UTC()
		query.From = &value
	}
	if request.To != nil {
		value := request.To.UTC()
		query.To = &value
	}
	return query
}

func dataSyncInvalidIssueTimeBounds(query data.DataSyncInvalidIssueQuery) (any, any) {
	var from any
	if query.From != nil {
		value := query.From.UTC()
		from = value
	}
	var to any
	if query.To != nil {
		value := query.To.UTC()
		to = value
	}
	return from, to
}
