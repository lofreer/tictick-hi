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

type BacktestTask struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Exchange       string         `json:"exchange"`
	Symbol         string         `json:"symbol"`
	Interval       string         `json:"interval"`
	StartTime      *time.Time     `json:"startTime,omitempty"`
	EndTime        *time.Time     `json:"endTime,omitempty"`
	RequestID      string         `json:"requestId,omitempty"`
	TraceParent    string         `json:"traceparent,omitempty"`
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
	RequestID      string         `json:"-"`
	TraceParent    string         `json:"-"`
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

type BacktestResult struct {
	TaskID        string
	Intents       []StrategyIntent
	Orders        []BacktestOrder
	ResultSummary map[string]any
}

type TradingTask struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Exchange       string         `json:"exchange"`
	AccountID      string         `json:"accountId"`
	Symbol         string         `json:"symbol"`
	Interval       string         `json:"interval"`
	StrategyID     string         `json:"strategyId"`
	StrategyParams map[string]any `json:"strategyParams"`
	IntentPolicy   map[string]any `json:"intentPolicy"`
	RequestID      string         `json:"requestId,omitempty"`
	TraceParent    string         `json:"traceparent,omitempty"`
	Status         TaskStatus     `json:"status"`
	LockedBy       string         `json:"lockedBy,omitempty"`
	LockedUntil    *time.Time     `json:"lockedUntil,omitempty"`
	HeartbeatAt    *time.Time     `json:"heartbeatAt,omitempty"`
	StartedAt      *time.Time     `json:"startedAt,omitempty"`
	FinishedAt     *time.Time     `json:"finishedAt,omitempty"`
	LastError      string         `json:"lastError,omitempty"`
	AttemptCount   int            `json:"attemptCount"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type CreateTradingTask struct {
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Exchange       string         `json:"exchange"`
	AccountID      string         `json:"accountId"`
	Symbol         string         `json:"symbol"`
	Interval       string         `json:"interval"`
	StrategyID     string         `json:"strategyId"`
	StrategyParams map[string]any `json:"strategyParams"`
	IntentPolicy   map[string]any `json:"intentPolicy"`
	RequestID      string         `json:"-"`
	TraceParent    string         `json:"-"`
}

type StrategyIntent struct {
	ID             string         `json:"id"`
	TaskID         string         `json:"taskId"`
	TaskType       string         `json:"taskType"`
	StrategyID     string         `json:"strategyId"`
	IntentType     string         `json:"intentType"`
	IdempotencyKey string         `json:"idempotencyKey"`
	Payload        map[string]any `json:"payload"`
	Policy         string         `json:"policy"`
	Status         string         `json:"status"`
	CreatedAt      time.Time      `json:"createdAt"`
}

type Order struct {
	ID                      string         `json:"id"`
	TaskID                  string         `json:"taskId"`
	TaskType                string         `json:"taskType"`
	IntentID                string         `json:"intentId,omitempty"`
	IdempotencyKey          string         `json:"idempotencyKey"`
	Exchange                string         `json:"exchange"`
	AccountID               string         `json:"accountId"`
	Symbol                  string         `json:"symbol"`
	Side                    string         `json:"side"`
	OrderType               string         `json:"orderType"`
	Price                   string         `json:"price"`
	Quantity                string         `json:"quantity"`
	Status                  string         `json:"status"`
	ExchangeOrderID         string         `json:"exchangeOrderId,omitempty"`
	ExchangeResponseSummary map[string]any `json:"exchangeResponseSummary"`
	LastError               string         `json:"lastError,omitempty"`
	CreatedAt               time.Time      `json:"createdAt"`
	UpdatedAt               time.Time      `json:"updatedAt"`
}

type Execution struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"taskId"`
	TaskType       string    `json:"taskType"`
	OrderID        string    `json:"orderId"`
	IntentID       string    `json:"intentId,omitempty"`
	IdempotencyKey string    `json:"idempotencyKey"`
	Exchange       string    `json:"exchange"`
	AccountID      string    `json:"accountId"`
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	Price          string    `json:"price"`
	Quantity       string    `json:"quantity"`
	Fee            string    `json:"fee"`
	Status         string    `json:"status"`
	ExecutedAt     time.Time `json:"executedAt"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Position struct {
	TaskID       string    `json:"taskId"`
	TaskType     string    `json:"taskType"`
	Exchange     string    `json:"exchange"`
	AccountID    string    `json:"accountId"`
	Symbol       string    `json:"symbol"`
	Quantity     string    `json:"quantity"`
	AveragePrice string    `json:"averagePrice"`
	RealizedPnL  string    `json:"realizedPnl"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Notification struct {
	ID                     string     `json:"id"`
	TaskID                 string     `json:"taskId,omitempty"`
	IntentID               string     `json:"intentId,omitempty"`
	RequestID              string     `json:"requestId,omitempty"`
	TraceParent            string     `json:"traceparent,omitempty"`
	Channel                string     `json:"channel"`
	Provider               string     `json:"provider"`
	ProviderMessageID      string     `json:"providerMessageId,omitempty"`
	Target                 string     `json:"target"`
	Title                  string     `json:"title"`
	Body                   string     `json:"body"`
	Status                 string     `json:"status"`
	Error                  string     `json:"error,omitempty"`
	AttemptCount           int        `json:"attemptCount"`
	MaxAttempts            int        `json:"maxAttempts"`
	NextAttemptAt          *time.Time `json:"nextAttemptAt,omitempty"`
	LastAttemptAt          *time.Time `json:"lastAttemptAt,omitempty"`
	LastDeliveryDurationMS *int64     `json:"lastDeliveryDurationMs,omitempty"`
	CreatedAt              time.Time  `json:"createdAt"`
	SentAt                 *time.Time `json:"sentAt,omitempty"`
}

type TradingRunResult struct {
	TaskID        string
	Intents       []StrategyIntent
	Orders        []Order
	Executions    []Execution
	Notifications []Notification
}

type NotificationDelivery struct {
	ID             string
	NotificationID string
	TaskID         string
	IntentID       string
	RequestID      string
	TraceParent    string
	Channel        string
	Provider       string
	Target         string
	Title          string
	Body           string
	Status         string
	AttemptCount   int
	MaxAttempts    int
	NextAttemptAt  time.Time
	LastAttemptAt  *time.Time
	LastError      string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type NotificationDeliveryResult struct {
	ProviderMessageID string
}

type NotificationChannel struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Provider  string    `json:"provider"`
	Target    string    `json:"target"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateNotificationChannel struct {
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Target   string `json:"target"`
	Enabled  bool   `json:"enabled"`
}

type ExchangeAccount struct {
	ID               string    `json:"id"`
	Exchange         string    `json:"exchange"`
	Alias            string    `json:"alias"`
	Enabled          bool      `json:"enabled"`
	CredentialStatus string    `json:"credentialStatus"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type CreateExchangeAccount struct {
	Exchange  string `json:"exchange"`
	Alias     string `json:"alias"`
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
	Enabled   bool   `json:"enabled"`
}

type Operator struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type CreateOperator struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role,omitempty"`
	Enabled  bool   `json:"enabled"`
}

type UpdateOperatorRole struct {
	Role string `json:"role"`
}

type OperatorRoleUpdateResult struct {
	Operator     Operator
	PreviousRole string
}

type OperatorSessionRevokeResult struct {
	RevokedSessionCount int `json:"revokedSessionCount"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ChangeOperatorPasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type ChangeOperatorPasswordResult struct {
	Status              string `json:"status"`
	RevokedSessionCount int    `json:"revokedSessionCount"`
}

type OperatorSession struct {
	ID                string    `json:"id"`
	OperatorID        string    `json:"operatorId,omitempty"`
	TokenHash         string    `json:"-"`
	RemoteAddr        string    `json:"remoteAddr,omitempty"`
	RemoteAddrChanged bool      `json:"remoteAddrChanged"`
	UserAgent         string    `json:"userAgent,omitempty"`
	UserAgentChanged  bool      `json:"userAgentChanged"`
	ExpiresAt         time.Time `json:"expiresAt"`
	CreatedAt         time.Time `json:"createdAt"`
	Current           bool      `json:"current"`
}

type SystemHealth struct {
	Status    string          `json:"status"`
	Database  string          `json:"database"`
	CheckedAt time.Time       `json:"checkedAt"`
	Services  []ServiceHealth `json:"services"`
}

type ServiceHealth struct {
	Name                   string     `json:"name"`
	Status                 string     `json:"status"`
	Detail                 string     `json:"detail,omitempty"`
	PendingCount           *int       `json:"pendingCount,omitempty"`
	RunningCount           *int       `json:"runningCount,omitempty"`
	LockedCount            *int       `json:"lockedCount,omitempty"`
	StaleLeaseCount        *int       `json:"staleLeaseCount,omitempty"`
	ExchangeBackoffCount   *int       `json:"exchangeBackoffCount,omitempty"`
	NextExchangeAttemptAt  *time.Time `json:"nextExchangeAttemptAt,omitempty"`
	FetchLockSkipCount     *int64     `json:"fetchLockSkipCount,omitempty"`
	LastFetchLockSkippedAt *time.Time `json:"lastFetchLockSkippedAt,omitempty"`
	LastHeartbeatAt        *time.Time `json:"lastHeartbeatAt,omitempty"`
	LockedUntil            *time.Time `json:"lockedUntil,omitempty"`
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

type CandleSource string

const (
	CandleSourceNative     CandleSource = "native"
	CandleSourceAggregated CandleSource = "aggregated"
	CandleSourceNone       CandleSource = "none"
)

type CandleHealth string

const (
	CandleHealthOK           CandleHealth = "ok"
	CandleHealthGap          CandleHealth = "gap"
	CandleHealthInsufficient CandleHealth = "insufficient"
	CandleHealthInvalid      CandleHealth = "invalid"
)

type CandleGap struct {
	From           time.Time `json:"from"`
	To             time.Time `json:"to"`
	MissingCandles int       `json:"missingCandles"`
}

type CandleIssue struct {
	Code     string     `json:"code"`
	Message  string     `json:"message"`
	OpenTime *time.Time `json:"openTime,omitempty"`
}

type CandleCoverage struct {
	RequestedLimit      int  `json:"requestedLimit"`
	ReturnedCandles     int  `json:"returnedCandles"`
	RequiredBaseCandles int  `json:"requiredBaseCandles,omitempty"`
	BaseLimit           int  `json:"baseLimit,omitempty"`
	ReturnedBaseCandles int  `json:"returnedBaseCandles,omitempty"`
	LimitedByBaseWindow bool `json:"limitedByBaseWindow"`
}

type CandleWindow struct {
	From  *time.Time `json:"from,omitempty"`
	To    *time.Time `json:"to,omitempty"`
	Count int        `json:"count"`
}

type CandlePagination struct {
	HasPrevious    bool       `json:"hasPrevious"`
	HasNext        bool       `json:"hasNext"`
	PreviousCursor string     `json:"previousCursor,omitempty"`
	NextCursor     string     `json:"nextCursor,omitempty"`
	PreviousFrom   *time.Time `json:"previousFrom,omitempty"`
	PreviousTo     *time.Time `json:"previousTo,omitempty"`
	NextFrom       *time.Time `json:"nextFrom,omitempty"`
	NextTo         *time.Time `json:"nextTo,omitempty"`
}

type CandleResult struct {
	Candles           []Candle         `json:"candles"`
	Source            CandleSource     `json:"source"`
	RequestedInterval string           `json:"requestedInterval"`
	BaseInterval      string           `json:"baseInterval,omitempty"`
	Health            CandleHealth     `json:"health"`
	Gaps              []CandleGap      `json:"gaps,omitempty"`
	Issues            []CandleIssue    `json:"issues,omitempty"`
	Coverage          CandleCoverage   `json:"coverage"`
	Window            CandleWindow     `json:"window"`
	Pagination        CandlePagination `json:"pagination"`
}

type NativeCandleStore interface {
	ListNativeCandles(ctx context.Context, query CandleQuery) ([]Candle, error)
	ListLatestNativeCandles(ctx context.Context, query CandleQuery) ([]Candle, error)
}

type SyncRepository interface {
	ClaimDataSyncTask(ctx context.Context, workerID string, leaseTTL time.Duration) (DataSyncTask, bool, error)
	HeartbeatDataSyncTask(ctx context.Context, taskID string, workerID string, leaseTTL time.Duration) error
	SaveDataSyncResult(ctx context.Context, result DataSyncResult) error
	RecordDataSyncRetry(ctx context.Context, taskID string, workerID string, err error, nextAttemptAt *time.Time) error
	MarkDataSyncFailed(ctx context.Context, taskID string, workerID string, err error) error
	ReleaseDataSyncTask(ctx context.Context, taskID string, workerID string) error
}

type BacktestRepository interface {
	ClaimBacktestTask(ctx context.Context, workerID string, leaseTTL time.Duration) (BacktestTask, bool, error)
	HeartbeatBacktestTask(ctx context.Context, taskID string, workerID string, leaseTTL time.Duration) error
	SaveBacktestResult(ctx context.Context, result BacktestResult) error
	MarkBacktestFailed(ctx context.Context, taskID string, err error) error
	ReleaseBacktestTask(ctx context.Context, taskID string) error
	GetCandles(ctx context.Context, query CandleQuery) (CandleResult, error)
}

type TradingRepository interface {
	ClaimTradingTask(ctx context.Context, workerID string, leaseTTL time.Duration) (TradingTask, bool, error)
	HeartbeatTradingTask(ctx context.Context, taskID string, workerID string, leaseTTL time.Duration) error
	SaveTradingRunResult(ctx context.Context, result TradingRunResult) error
	MarkTradingTaskFailed(ctx context.Context, taskID string, err error) error
	ReleaseTradingTask(ctx context.Context, taskID string) error
	GetCandles(ctx context.Context, query CandleQuery) (CandleResult, error)
}

type NotificationRepository interface {
	ClaimNotificationDelivery(ctx context.Context, workerID string, leaseTTL time.Duration) (NotificationDelivery, bool, error)
	MarkNotificationDelivered(ctx context.Context, deliveryID string, deliveredAt time.Time, result NotificationDeliveryResult, deliveryDuration time.Duration) error
	MarkNotificationFailed(ctx context.Context, deliveryID string, err error, nextAttemptAt *time.Time, deliveryDuration time.Duration) error
	ReleaseNotificationDelivery(ctx context.Context, deliveryID string) error
}
