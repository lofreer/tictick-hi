package postgres

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/lofreer/tictick-hi/internal/data"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func dataSyncTaskReturningColumns() string {
	return dataSyncTaskScanColumns("", dataSyncTaskStateHealthSQL(""), dataSyncTaskEmptyGapSummarySQL())
}

func dataSyncTaskScanColumns(alias string, healthSQL string, gapSummarySQL string) string {
	return fmt.Sprintf(`%s, %s AS data_health, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s`,
		dataSyncTaskColumnList(alias, "id", "exchange", "symbol", "interval", "start_time", "end_time",
			"sync_enabled", "realtime_enabled", "status"),
		healthSQL,
		gapSummarySQL,
		fmt.Sprintf("COALESCE(%s, '')", dataSyncTaskColumn(alias, "repair_source_task_id")),
		dataSyncTaskColumn(alias, "last_synced_open_time"),
		fmt.Sprintf("COALESCE(%s, '')", dataSyncTaskColumn(alias, "last_error")),
		dataSyncTaskColumn(alias, "attempt_count"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskExchangeBackoffUntilSQL(alias),
		dataSyncTaskExchangeBackoffLastErrorSQL(alias),
		dataSyncTaskColumn(alias, "created_at"),
		dataSyncTaskColumn(alias, "updated_at"),
	)
}

func dataSyncTaskListGapSummarySQL() string {
	return `COALESCE(candle_state.gap_count, 0) AS gap_count,
		candle_state.first_gap_from AS first_gap_from,
		candle_state.first_gap_to AS first_gap_to,
		COALESCE(candle_state.first_gap_missing_candles, 0) AS first_gap_missing_candles`
}

func dataSyncTaskEmptyGapSummarySQL() string {
	return `0 AS gap_count,
		NULL::timestamptz AS first_gap_from,
		NULL::timestamptz AS first_gap_to,
		0 AS first_gap_missing_candles`
}

func dataSyncTaskColumnList(alias string, columns ...string) string {
	result := ""
	for index, column := range columns {
		if index > 0 {
			result += ", "
		}
		result += dataSyncTaskColumn(alias, column)
	}
	return result
}

func dataSyncTaskColumn(alias string, column string) string {
	if alias == "" {
		return column
	}
	return alias + "." + column
}

func dataSyncTaskListHealthSQL(alias string) string {
	return fmt.Sprintf(`CASE
		WHEN %s IN ('failed', 'cancelled') THEN 'failed'
		WHEN %s IS NOT NULL AND %s > now() THEN 'retrying'
		WHEN %s THEN 'retrying'
		WHEN candle_state.has_gap THEN 'gap'
		WHEN %s = 'paused' THEN 'paused'
		WHEN %s IS NULL OR COALESCE(candle_state.candle_count, 0) = 0 THEN
			CASE WHEN (%s IN ('pending', 'running') AND (%s OR %s)) THEN 'syncing' ELSE 'insufficient' END
		WHEN %s IN ('pending', 'running') AND (%s OR %s) THEN 'syncing'
		ELSE 'ok'
	END`,
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskExchangeBackoffActiveSQL(alias),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "last_synced_open_time"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
	)
}

func dataSyncTaskStateHealthSQL(alias string) string {
	return fmt.Sprintf(`CASE
		WHEN %s IN ('failed', 'cancelled') THEN 'failed'
		WHEN %s IS NOT NULL AND %s > now() THEN 'retrying'
		WHEN %s THEN 'retrying'
		WHEN %s = 'paused' THEN 'paused'
		WHEN %s IS NULL THEN
			CASE WHEN (%s IN ('pending', 'running') AND (%s OR %s)) THEN 'syncing' ELSE 'insufficient' END
		WHEN %s IN ('pending', 'running') AND (%s OR %s) THEN 'syncing'
		ELSE 'ok'
	END`,
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskColumn(alias, "next_attempt_at"),
		dataSyncTaskExchangeBackoffActiveSQL(alias),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "last_synced_open_time"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
		dataSyncTaskColumn(alias, "status"),
		dataSyncTaskColumn(alias, "sync_enabled"),
		dataSyncTaskColumn(alias, "realtime_enabled"),
	)
}

func dataSyncTaskIntervalDurationSQL(alias string) string {
	intervalColumn := dataSyncTaskColumn(alias, "interval")
	return fmt.Sprintf(`CASE %s
		WHEN '1m' THEN interval '1 minute'
		WHEN '5m' THEN interval '5 minutes'
		WHEN '15m' THEN interval '15 minutes'
		WHEN '1h' THEN interval '1 hour'
		WHEN '4h' THEN interval '4 hours'
		WHEN '1d' THEN interval '1 day'
		ELSE NULL
	END`, intervalColumn)
}

func dataSyncTaskExchangeBackoffActiveSQL(alias string) string {
	return fmt.Sprintf(`EXISTS (
		SELECT 1
		  FROM data_sync_exchange_backoffs AS exchange_backoff
		 WHERE exchange_backoff.exchange = %s
		   AND exchange_backoff.next_attempt_at > now()
	)`, dataSyncTaskColumn(alias, "exchange"))
}

func dataSyncTaskExchangeBackoffUntilSQL(alias string) string {
	return fmt.Sprintf(`(
		SELECT exchange_backoff.next_attempt_at
		  FROM data_sync_exchange_backoffs AS exchange_backoff
		 WHERE exchange_backoff.exchange = %s
		   AND exchange_backoff.next_attempt_at > now()
		 LIMIT 1
	) AS exchange_backoff_until`, dataSyncTaskColumn(alias, "exchange"))
}

func dataSyncTaskExchangeBackoffLastErrorSQL(alias string) string {
	return fmt.Sprintf(`COALESCE((
		SELECT exchange_backoff.last_error
		  FROM data_sync_exchange_backoffs AS exchange_backoff
		 WHERE exchange_backoff.exchange = %s
		   AND exchange_backoff.next_attempt_at > now()
		 LIMIT 1
	), '') AS exchange_backoff_last_error`, dataSyncTaskColumn(alias, "exchange"))
}

func scanDataSyncTask(row pgx.CollectableRow) (data.DataSyncTask, error) {
	return scanDataSyncTaskRow(row)
}

func scanDataSyncTaskRow(row rowScanner) (data.DataSyncTask, error) {
	var task data.DataSyncTask
	var gapCount int64
	var firstGapFrom sql.NullTime
	var firstGapTo sql.NullTime
	var firstGapMissingCandles sql.NullInt64
	err := row.Scan(
		&task.ID,
		&task.Exchange,
		&task.Symbol,
		&task.Interval,
		&task.StartTime,
		&task.EndTime,
		&task.SyncEnabled,
		&task.RealtimeEnabled,
		&task.Status,
		&task.DataHealth,
		&gapCount,
		&firstGapFrom,
		&firstGapTo,
		&firstGapMissingCandles,
		&task.RepairSourceTaskID,
		&task.LatestSyncedOpenTime,
		&task.LastError,
		&task.AttemptCount,
		&task.NextAttemptAt,
		&task.ExchangeBackoffUntil,
		&task.ExchangeBackoffError,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err == nil && gapCount > 0 {
		task.GapSummary = &data.DataSyncGapSummary{Count: int(gapCount)}
		if firstGapFrom.Valid && firstGapTo.Valid {
			task.GapSummary.FirstGap = &data.CandleGap{
				From:           firstGapFrom.Time,
				To:             firstGapTo.Time,
				MissingCandles: int(firstGapMissingCandles.Int64),
			}
		}
	}
	return task, err
}
