package data

import "time"

type DataSyncHealth string

const (
	DataSyncHealthOK           DataSyncHealth = "ok"
	DataSyncHealthSyncing      DataSyncHealth = "syncing"
	DataSyncHealthGap          DataSyncHealth = "gap"
	DataSyncHealthFailed       DataSyncHealth = "failed"
	DataSyncHealthPaused       DataSyncHealth = "paused"
	DataSyncHealthRetrying     DataSyncHealth = "retrying"
	DataSyncHealthInsufficient DataSyncHealth = "insufficient"
)

type DataSyncTask struct {
	ID                   string              `json:"id"`
	Exchange             string              `json:"exchange"`
	Symbol               string              `json:"symbol"`
	Interval             string              `json:"interval"`
	StartTime            *time.Time          `json:"startTime,omitempty"`
	EndTime              *time.Time          `json:"endTime,omitempty"`
	SyncEnabled          bool                `json:"syncEnabled"`
	RealtimeEnabled      bool                `json:"realtimeEnabled"`
	Status               TaskStatus          `json:"status"`
	DataHealth           DataSyncHealth      `json:"dataHealth"`
	GapSummary           *DataSyncGapSummary `json:"gapSummary,omitempty"`
	LatestSyncedOpenTime *time.Time          `json:"latestSyncedAt,omitempty"`
	LastError            string              `json:"lastError,omitempty"`
	AttemptCount         int                 `json:"attemptCount"`
	NextAttemptAt        *time.Time          `json:"nextAttemptAt,omitempty"`
	CreatedAt            time.Time           `json:"createdAt"`
	UpdatedAt            time.Time           `json:"updatedAt"`
}

type DataSyncGapSummary struct {
	Count    int        `json:"count"`
	FirstGap *CandleGap `json:"firstGap,omitempty"`
}

type CreateDataSyncTask struct {
	Exchange  string     `json:"exchange"`
	Symbol    string     `json:"symbol"`
	Interval  string     `json:"interval"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
}
