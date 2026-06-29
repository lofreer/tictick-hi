package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

const maxDataSyncInvalidIssues = 50

func (store *Store) ListDataSyncTaskInvalidIssues(ctx context.Context, id string) (data.DataSyncInvalidIssueList, error) {
	task, err := getDataSyncTask(ctx, store.pool, id)
	if err != nil {
		return data.DataSyncInvalidIssueList{}, err
	}

	issues, totalCount, limited, err := listDataSyncInvalidIssues(ctx, store.pool, id)
	if err != nil {
		return data.DataSyncInvalidIssueList{}, err
	}
	return data.DataSyncInvalidIssueList{
		TaskID:        task.ID,
		Issues:        issues,
		Limited:       limited,
		TotalCount:    totalCount,
		ReturnedCount: len(issues),
		IssueLimit:    maxDataSyncInvalidIssues,
	}, nil
}

func listDataSyncInvalidIssues(
	ctx context.Context,
	queryer dataSyncGapQueryer,
	id string,
) ([]data.CandleIssue, int, bool, error) {
	rows, err := queryer.Query(ctx, fmt.Sprintf(`
		WITH %s
		SELECT open_time, invalid_code, invalid_message, COUNT(*) OVER () AS total_count
		  FROM ordered_candles
		 WHERE invalid_code IS NOT NULL
		 ORDER BY open_time
		 LIMIT $2`,
		dataSyncTaskWindowGapCTESQL(`
			SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time
			  FROM data_sync_tasks AS t
			 WHERE t.id = $1
			   AND t.deleted_at IS NULL`),
	), id, maxDataSyncInvalidIssues)
	if err != nil {
		return nil, 0, false, fmt.Errorf("list data sync invalid issues: %w", err)
	}
	defer rows.Close()

	issues := make([]data.CandleIssue, 0)
	totalCount := 0
	for rows.Next() {
		var openTime time.Time
		var issue data.CandleIssue
		if err := rows.Scan(&openTime, &issue.Code, &issue.Message, &totalCount); err != nil {
			return nil, 0, false, fmt.Errorf("scan data sync invalid issue: %w", err)
		}
		issue.OpenTime = &openTime
		issues = append(issues, issue)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, false, fmt.Errorf("iterate data sync invalid issues: %w", err)
	}

	limited := totalCount > maxDataSyncInvalidIssues
	return issues, totalCount, limited, nil
}
