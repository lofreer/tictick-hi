package data

import "time"

const (
	DefaultOverviewRecentFactLimit = 8
	MaxOverviewRecentFactLimit     = 50
	DefaultOverviewTrendDays       = 7
	MaxOverviewTrendDays           = 30
)

type OverviewRecentFacts struct {
	StrategyIntents []OverviewStrategyIntentFact `json:"strategyIntents"`
	Orders          []OverviewOrderFact          `json:"orders"`
}

type OverviewRecentFactQuery struct {
	Limit int
	Since *time.Time
}

type OverviewTrends struct {
	Days    int                   `json:"days"`
	From    time.Time             `json:"from"`
	To      time.Time             `json:"to"`
	Buckets []OverviewTrendBucket `json:"buckets"`
}

type OverviewTrendQuery struct {
	Days int
	From time.Time
	To   time.Time
}

type OverviewTrendBucket struct {
	BucketStart     time.Time `json:"bucketStart"`
	StrategyIntents int       `json:"strategyIntents"`
	Orders          int       `json:"orders"`
	Notifications   int       `json:"notifications"`
	Failures        int       `json:"failures"`
}

type OverviewStrategyIntentFact struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	TaskType   string    `json:"taskType"`
	TaskName   string    `json:"taskName"`
	Exchange   string    `json:"exchange"`
	Symbol     string    `json:"symbol"`
	Interval   string    `json:"interval"`
	StrategyID string    `json:"strategyId"`
	IntentType string    `json:"intentType"`
	Policy     string    `json:"policy"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type OverviewOrderFact struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"taskId"`
	TaskType   string    `json:"taskType"`
	TaskName   string    `json:"taskName"`
	Exchange   string    `json:"exchange"`
	Symbol     string    `json:"symbol"`
	Interval   string    `json:"interval"`
	IntentID   string    `json:"intentId,omitempty"`
	Side       string    `json:"side"`
	Price      string    `json:"price"`
	Quantity   string    `json:"quantity"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurredAt"`
}
