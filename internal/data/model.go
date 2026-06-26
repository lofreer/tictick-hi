package data

import (
	"context"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusStopping  TaskStatus = "stopping"
	TaskStatusPaused    TaskStatus = "paused"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type DataSyncTask struct {
	ID                   string     `json:"id"`
	Exchange             string     `json:"exchange"`
	Symbol               string     `json:"symbol"`
	Interval             string     `json:"interval"`
	StartTime            *time.Time `json:"startTime,omitempty"`
	EndTime              *time.Time `json:"endTime,omitempty"`
	SyncEnabled          bool       `json:"syncEnabled"`
	RealtimeEnabled      bool       `json:"realtimeEnabled"`
	Status               TaskStatus `json:"status"`
	LatestSyncedOpenTime *time.Time `json:"latestSyncedAt,omitempty"`
	LastError            string     `json:"lastError,omitempty"`
	AttemptCount         int        `json:"attemptCount"`
	CreatedAt            time.Time  `json:"createdAt"`
	UpdatedAt            time.Time  `json:"updatedAt"`
}

type CreateDataSyncTask struct {
	Exchange  string     `json:"exchange"`
	Symbol    string     `json:"symbol"`
	Interval  string     `json:"interval"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
}

type Candle struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	OpenTime  time.Time `json:"openTime"`
	CloseTime time.Time `json:"closeTime"`
	Open      string    `json:"open"`
	High      string    `json:"high"`
	Low       string    `json:"low"`
	Close     string    `json:"close"`
	Volume    string    `json:"volume"`
	IsClosed  bool      `json:"isClosed"`
}

type CandleQuery struct {
	Exchange string
	Symbol   string
	Interval string
	From     *time.Time
	To       *time.Time
	Limit    int
}

type Repository interface {
	ListDataSyncTasks(ctx context.Context) ([]DataSyncTask, error)
	CreateDataSyncTask(ctx context.Context, task CreateDataSyncTask) (DataSyncTask, error)
	DeleteDataSyncTask(ctx context.Context, id string) error
	SetSyncEnabled(ctx context.Context, id string, enabled bool) (DataSyncTask, error)
	SetRealtimeEnabled(ctx context.Context, id string, enabled bool) (DataSyncTask, error)
	ListCandles(ctx context.Context, query CandleQuery) ([]Candle, error)
}

type SyncRepository interface {
	ClaimDataSyncTask(ctx context.Context, workerID string, leaseTTL time.Duration) (DataSyncTask, bool, error)
	SaveDataSyncResult(ctx context.Context, result DataSyncResult) error
	MarkDataSyncFailed(ctx context.Context, taskID string, err error) error
}

type DataSyncResult struct {
	TaskID       string
	Candles      []Candle
	LastOpenTime *time.Time
	Completed    bool
}
