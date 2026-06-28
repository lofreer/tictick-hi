package data

import "time"

type DataSyncHealth string
type DataSyncMarketStatus string

const (
	DataSyncHealthOK           DataSyncHealth = "ok"
	DataSyncHealthSyncing      DataSyncHealth = "syncing"
	DataSyncHealthGap          DataSyncHealth = "gap"
	DataSyncHealthFailed       DataSyncHealth = "failed"
	DataSyncHealthPaused       DataSyncHealth = "paused"
	DataSyncHealthRetrying     DataSyncHealth = "retrying"
	DataSyncHealthInsufficient DataSyncHealth = "insufficient"
)

const (
	DataSyncMarketStatusActive   DataSyncMarketStatus = "active"
	DataSyncMarketStatusInactive DataSyncMarketStatus = "inactive"
	DataSyncMarketStatusMissing  DataSyncMarketStatus = "missing"
)

type DataSyncTask struct {
	ID                   string               `json:"id"`
	Exchange             string               `json:"exchange"`
	Symbol               string               `json:"symbol"`
	Interval             string               `json:"interval"`
	StartTime            *time.Time           `json:"startTime,omitempty"`
	EndTime              *time.Time           `json:"endTime,omitempty"`
	RepairSourceTaskID   string               `json:"repairSourceTaskId,omitempty"`
	SyncEnabled          bool                 `json:"syncEnabled"`
	RealtimeEnabled      bool                 `json:"realtimeEnabled"`
	Status               TaskStatus           `json:"status"`
	MarketStatus         DataSyncMarketStatus `json:"marketStatus"`
	DataHealth           DataSyncHealth       `json:"dataHealth"`
	GapSummary           *DataSyncGapSummary  `json:"gapSummary,omitempty"`
	LatestSyncedOpenTime *time.Time           `json:"latestSyncedAt,omitempty"`
	LastError            string               `json:"lastError,omitempty"`
	AttemptCount         int                  `json:"attemptCount"`
	NextAttemptAt        *time.Time           `json:"nextAttemptAt,omitempty"`
	ExchangeBackoffUntil *time.Time           `json:"exchangeBackoffUntil,omitempty"`
	ExchangeBackoffError string               `json:"exchangeBackoffLastError,omitempty"`
	CreatedAt            time.Time            `json:"createdAt"`
	UpdatedAt            time.Time            `json:"updatedAt"`
}

type DataSyncGapSummary struct {
	Count    int        `json:"count"`
	FirstGap *CandleGap `json:"firstGap,omitempty"`
}

type DataSyncGapList struct {
	TaskID        string      `json:"taskId"`
	Gaps          []CandleGap `json:"gaps"`
	Limited       bool        `json:"limited"`
	TotalCount    int         `json:"totalCount"`
	ReturnedCount int         `json:"returnedCount"`
	RepairLimit   int         `json:"repairLimit"`
}

type DataSyncGapRepairResult struct {
	SourceTaskID    string         `json:"sourceTaskId"`
	CreatedTasks    []DataSyncTask `json:"createdTasks"`
	SkippedExisting int            `json:"skippedExisting"`
	Limited         bool           `json:"limited"`
	TotalCount      int            `json:"totalCount"`
	RepairLimit     int            `json:"repairLimit"`
}

type RepairDataSyncTaskGapRequest struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type CreateDataSyncTask struct {
	Exchange  string     `json:"exchange"`
	Symbol    string     `json:"symbol"`
	Interval  string     `json:"interval"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
}
