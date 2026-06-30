package data

import (
	"strings"
	"testing"
	"time"
)

func TestValidateDataSyncTaskWindow(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(time.Minute)

	cases := []struct {
		name      string
		interval  string
		startTime *time.Time
		endTime   *time.Time
		wantErr   string
	}{
		{name: "unbounded valid interval", interval: "1m"},
		{name: "single-sided start", interval: "1m", startTime: &start},
		{name: "single-sided end", interval: "1m", endTime: &end},
		{name: "bounded", interval: "1m", startTime: &start, endTime: &end},
		{name: "equal window", interval: "1m", startTime: &start, endTime: &start, wantErr: "startTime must be before endTime"},
		{name: "reversed window", interval: "1m", startTime: &end, endTime: &start, wantErr: "startTime must be before endTime"},
		{name: "unsupported interval", interval: "2m", wantErr: `unsupported data sync interval "2m"`},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateDataSyncTaskWindow(testCase.interval, testCase.startTime, testCase.endTime)
			if testCase.wantErr == "" {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), testCase.wantErr) {
				t.Fatalf("err = %v, want %q", err, testCase.wantErr)
			}
		})
	}
}
