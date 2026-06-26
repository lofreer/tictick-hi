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

type BacktestTask struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Exchange       string         `json:"exchange"`
	Symbol         string         `json:"symbol"`
	Interval       string         `json:"interval"`
	StartTime      *time.Time     `json:"startTime,omitempty"`
	EndTime        *time.Time     `json:"endTime,omitempty"`
	StrategyID     string         `json:"strategyId"`
	StrategyParams map[string]any `json:"strategyParams"`
	InitialBalance string         `json:"initialBalance"`
	FeeBps         string         `json:"feeBps"`
	SlippageBps    string         `json:"slippageBps"`
	TriggerMode    string         `json:"triggerMode"`
	Status         TaskStatus     `json:"status"`
	StartedAt      *time.Time     `json:"startedAt,omitempty"`
	FinishedAt     *time.Time     `json:"finishedAt,omitempty"`
	LastError      string         `json:"lastError,omitempty"`
	AttemptCount   int            `json:"attemptCount"`
	ResultSummary  map[string]any `json:"resultSummary"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type CreateBacktestTask struct {
	Name           string         `json:"name"`
	Exchange       string         `json:"exchange"`
	Symbol         string         `json:"symbol"`
	Interval       string         `json:"interval"`
	StartTime      *time.Time     `json:"startTime,omitempty"`
	EndTime        *time.Time     `json:"endTime,omitempty"`
	StrategyID     string         `json:"strategyId"`
	StrategyParams map[string]any `json:"strategyParams"`
	InitialBalance string         `json:"initialBalance"`
	FeeBps         string         `json:"feeBps"`
	SlippageBps    string         `json:"slippageBps"`
	TriggerMode    string         `json:"triggerMode"`
}

type BacktestOrder struct {
	ID         string    `json:"id"`
	BacktestID string    `json:"backtestId"`
	IntentID   string    `json:"intentId,omitempty"`
	Side       string    `json:"side"`
	Price      string    `json:"price"`
	Quantity   string    `json:"quantity"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurredAt"`
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
	ListBacktestTasks(ctx context.Context) ([]BacktestTask, error)
	CreateBacktestTask(ctx context.Context, task CreateBacktestTask) (BacktestTask, error)
	GetBacktestTask(ctx context.Context, id string) (BacktestTask, error)
	ListBacktestOrders(ctx context.Context, backtestID string) ([]BacktestOrder, error)
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
