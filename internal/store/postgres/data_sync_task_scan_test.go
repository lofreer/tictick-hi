package postgres

import (
	"strings"
	"testing"
)

func TestDataSyncTaskScanColumnsPlaceMarketStatusBeforeHealth(t *testing.T) {
	columns := dataSyncTaskScanColumns("t", dataSyncTaskListHealthSQL("t"), dataSyncTaskListGapSummarySQL())

	if strings.Contains(columns, "AS market_status AS data_health") {
		t.Fatal("data sync task scan columns double-aliased market status as data health")
	}
	if strings.Count(columns, "AS market_status") != 1 {
		t.Fatalf("market status alias count = %d, want 1", strings.Count(columns, "AS market_status"))
	}
	if strings.Index(columns, "AS market_status") > strings.Index(columns, "AS data_health") {
		t.Fatal("market status must be scanned before data health")
	}
}
