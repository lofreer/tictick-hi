package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const maxDataSyncInvalidIssues = data.MaxDataSyncInvalidIssueLimit

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

func listDataSyncInvalidIssues(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	id string,
	query data.DataSyncInvalidIssueQuery,
) ([]data.CandleIssue, int, error) {
	var totalCount int
	if err := queryer.QueryRow(ctx, fmt.Sprintf(`
		WITH %s
		SELECT COUNT(*)
		  FROM ordered_candles
		 WHERE invalid_code IS NOT NULL`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
	), id).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("count data sync invalid issues: %w", err)
	}
	if totalCount == 0 || query.Offset >= totalCount {
		return []data.CandleIssue{}, totalCount, nil
	}

	rows, err := queryer.Query(ctx, fmt.Sprintf(`
		WITH %s
		SELECT open_time, invalid_code, invalid_message
		  FROM ordered_candles
		 WHERE invalid_code IS NOT NULL
		 ORDER BY open_time
		 LIMIT $2 OFFSET $3`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
	), id, query.Limit, query.Offset)
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
