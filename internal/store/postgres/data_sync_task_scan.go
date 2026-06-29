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
	return fmt.Sprintf(`%s, %s, %s, %s AS data_health, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s`,
		dataSyncTaskColumnList(alias, "id", "exchange", "symbol", "interval", "start_time", "end_time",
			"sync_enabled", "realtime_enabled", "status"),
		dataSyncTaskMarketStatusSQL(alias),
		dataSyncTaskMarketStatusDetailSQL(alias),
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

func dataSyncTaskMarketStatusDetailSQL(alias string) string {
	return fmt.Sprintf(`COALESCE((
		SELECT instrument.exchange_status
		  FROM market_instruments AS instrument
		 WHERE instrument.exchange = %s
		   AND instrument.symbol = %s
		 LIMIT 1
	), 'missing') AS market_status_detail`,
		dataSyncTaskOuterColumn(alias, "exchange"),
		dataSyncTaskOuterColumn(alias, "symbol"),
	)
}

func dataSyncTaskMarketStatusSQL(alias string) string {
	return fmt.Sprintf(`COALESCE((
		SELECT CASE
		         WHEN instrument.status = 'active' THEN 'active'
		         ELSE 'inactive'
		       END
		  FROM market_instruments AS instrument
		 WHERE instrument.exchange = %s
		   AND instrument.symbol = %s
		 LIMIT 1
	), 'missing') AS market_status`,
		dataSyncTaskOuterColumn(alias, "exchange"),
		dataSyncTaskOuterColumn(alias, "symbol"),
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

func dataSyncTaskCandleStateLateralSQL() string {
	return fmt.Sprintf(`
		WITH %s
		SELECT (SELECT candle_count FROM candle_stats) AS candle_count,
		       (SELECT invalid_count FROM candle_stats) AS invalid_count,
		       COUNT(*) AS gap_count,
		       (ARRAY_AGG(gap_from ORDER BY gap_from, gap_to))[1] AS first_gap_from,
		       (ARRAY_AGG(gap_to ORDER BY gap_from, gap_to))[1] AS first_gap_to,
		       (ARRAY_AGG(missing_candles ORDER BY gap_from, gap_to))[1] AS first_gap_missing_candles,
		       COUNT(*) > 0 AS has_gap,
		       (SELECT invalid_count > 0 FROM candle_stats) AS has_invalid
		  FROM gaps`,
		dataSyncTaskWindowGapCTESQL(`SELECT t.exchange, t.symbol, t.interval, t.start_time, t.end_time, t.last_synced_open_time`),
	)
}

func dataSyncTaskWindowGapCTESQL(taskSourceSQL string) string {
	return fmt.Sprintf(`
		task_source AS (
			%s
		),
		task_base AS (
			SELECT task_source.*,
			       %s AS interval_duration,
			       COALESCE(task_source.end_time, task_source.last_synced_open_time, now()) AS scan_end,
			       COALESCE(task_source.end_time, task_source.last_synced_open_time) AS gap_window_end
			  FROM task_source
		),
		task_aligned AS (
			SELECT task_base.*,
			       date_bin(task_base.interval_duration, task_base.start_time, TIMESTAMPTZ '1970-01-01 00:00:00+00') AS start_floor,
			       date_bin(task_base.interval_duration, task_base.gap_window_end, TIMESTAMPTZ '1970-01-01 00:00:00+00') AS end_floor
			  FROM task_base
		),
		task_window AS (
			SELECT task_aligned.*,
			       CASE
			         WHEN task_aligned.start_time IS NULL OR task_aligned.interval_duration IS NULL THEN NULL
			         WHEN task_aligned.start_floor < task_aligned.start_time THEN task_aligned.start_floor + task_aligned.interval_duration
			         ELSE task_aligned.start_floor
			       END AS first_expected_open,
			       task_aligned.end_floor AS last_expected_open
			  FROM task_aligned
		),
		ordered_candles AS (
			SELECT c.open_time,
			       c.open,
			       c.high,
			       c.low,
			       c.close,
			       c.volume,
			       LAG(c.open_time) OVER (ORDER BY c.open_time) AS previous_open_time,
			       task_window.interval_duration
			  FROM task_window
			  JOIN market_candles AS c
			    ON c.exchange = task_window.exchange
			   AND c.symbol = task_window.symbol
			   AND c.interval = task_window.interval
			 WHERE (task_window.start_time IS NULL OR c.open_time >= task_window.start_time)
			   AND c.open_time <= task_window.scan_end
		),
		candle_stats AS (
			SELECT COUNT(*)::int AS candle_count,
			       COUNT(*) FILTER (
			         WHERE open <= 0
			            OR high <= 0
			            OR low <= 0
			            OR close <= 0
			            OR volume < 0
			            OR high < GREATEST(open, close, low)
			            OR low > LEAST(open, close, high)
			       )::int AS invalid_count,
			       MIN(open_time) AS first_open,
			       MAX(open_time) AS last_open
			  FROM ordered_candles
		),
		gaps AS (
			SELECT task_window.first_expected_open AS gap_from,
			       candle_stats.first_open AS gap_to,
			       GREATEST(
			         (EXTRACT(EPOCH FROM (candle_stats.first_open - task_window.first_expected_open))
			          / NULLIF(EXTRACT(EPOCH FROM task_window.interval_duration), 0))::int,
			         1
			       ) AS missing_candles
			  FROM task_window
			  CROSS JOIN candle_stats
			 WHERE task_window.first_expected_open IS NOT NULL
			   AND candle_stats.first_open IS NOT NULL
			   AND candle_stats.first_open > task_window.first_expected_open
			UNION ALL
			SELECT ordered_candles.previous_open_time + ordered_candles.interval_duration AS gap_from,
			       ordered_candles.open_time AS gap_to,
			       GREATEST(
			         (EXTRACT(EPOCH FROM (ordered_candles.open_time - (ordered_candles.previous_open_time + ordered_candles.interval_duration)))
			          / NULLIF(EXTRACT(EPOCH FROM ordered_candles.interval_duration), 0))::int,
			         1
			       ) AS missing_candles
			  FROM ordered_candles
			 WHERE ordered_candles.previous_open_time IS NOT NULL
			   AND ordered_candles.interval_duration IS NOT NULL
			   AND ordered_candles.open_time - ordered_candles.previous_open_time > ordered_candles.interval_duration
			UNION ALL
			SELECT candle_stats.last_open + task_window.interval_duration AS gap_from,
			       task_window.last_expected_open + task_window.interval_duration AS gap_to,
			       GREATEST(
			         (EXTRACT(EPOCH FROM ((task_window.last_expected_open + task_window.interval_duration) - (candle_stats.last_open + task_window.interval_duration)))
			          / NULLIF(EXTRACT(EPOCH FROM task_window.interval_duration), 0))::int,
			         1
			       ) AS missing_candles
			  FROM task_window
			  CROSS JOIN candle_stats
			 WHERE task_window.last_expected_open IS NOT NULL
			   AND candle_stats.last_open IS NOT NULL
			   AND task_window.last_expected_open >= candle_stats.last_open + task_window.interval_duration
			UNION ALL
			SELECT task_window.first_expected_open AS gap_from,
			       task_window.last_expected_open + task_window.interval_duration AS gap_to,
			       GREATEST(
			         (EXTRACT(EPOCH FROM ((task_window.last_expected_open + task_window.interval_duration) - task_window.first_expected_open))
			          / NULLIF(EXTRACT(EPOCH FROM task_window.interval_duration), 0))::int,
			         1
			       ) AS missing_candles
			  FROM task_window
			  CROSS JOIN candle_stats
			 WHERE candle_stats.candle_count = 0
			   AND task_window.first_expected_open IS NOT NULL
			   AND task_window.last_expected_open IS NOT NULL
			   AND task_window.last_expected_open >= task_window.first_expected_open
		)`,
		taskSourceSQL,
		dataSyncTaskIntervalDurationSQL("task_source"),
	)
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

func dataSyncTaskOuterColumn(alias string, column string) string {
	if alias == "" {
		return "data_sync_tasks." + column
	}
	return dataSyncTaskColumn(alias, column)
}

func dataSyncTaskListHealthSQL(alias string) string {
	return fmt.Sprintf(`CASE
		WHEN %s IN ('failed', 'cancelled') THEN 'failed'
		WHEN %s IS NOT NULL AND %s > now() THEN 'retrying'
		WHEN %s THEN 'retrying'
		WHEN COALESCE(candle_state.has_invalid, false) THEN 'invalid'
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
		&task.MarketStatus,
		&task.MarketStatusDetail,
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
