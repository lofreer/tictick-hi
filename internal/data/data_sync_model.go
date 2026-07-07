package data

import "time"

type DataSyncHealth string
type DataSyncMarketStatus string

var candleIssueCodes = []string{
	CandleIssueInvalidOpenPrice,
	CandleIssueInvalidHighPrice,
	CandleIssueInvalidLowPrice,
	CandleIssueInvalidClosePrice,
	CandleIssueInvalidVolume,
	CandleIssueInvalidHighBound,
	CandleIssueInvalidLowBound,
	CandleIssueInvalidOpenTime,
	CandleIssueInvalidCloseTime,
}

const (
	DataSyncHealthOK           DataSyncHealth = "ok"
	DataSyncHealthSyncing      DataSyncHealth = "syncing"
	DataSyncHealthGap          DataSyncHealth = "gap"
	DataSyncHealthFailed       DataSyncHealth = "failed"
	DataSyncHealthPaused       DataSyncHealth = "paused"
	DataSyncHealthRetrying     DataSyncHealth = "retrying"
	DataSyncHealthInsufficient DataSyncHealth = "insufficient"
	DataSyncHealthInvalid      DataSyncHealth = "invalid"
)

const (
	DataSyncMarketStatusActive   DataSyncMarketStatus = "active"
	DataSyncMarketStatusInactive DataSyncMarketStatus = "inactive"
	DataSyncMarketStatusMissing  DataSyncMarketStatus = "missing"
)

const (
	CandleIssueInvalidOpenPrice  = "invalid_open_price"
	CandleIssueInvalidHighPrice  = "invalid_high_price"
	CandleIssueInvalidLowPrice   = "invalid_low_price"
	CandleIssueInvalidClosePrice = "invalid_close_price"
	CandleIssueInvalidVolume     = "invalid_volume"
	CandleIssueInvalidHighBound  = "invalid_high_bound"
	CandleIssueInvalidLowBound   = "invalid_low_bound"
	CandleIssueInvalidOpenTime   = "invalid_open_time"
	CandleIssueInvalidCloseTime  = "invalid_close_time"
)

const (
	DefaultDataSyncInvalidIssueLimit = 50
	MaxDataSyncInvalidIssueLimit     = 50
)

type DataSyncTask struct {
	ID                   string                  `json:"id"`
	Exchange             string                  `json:"exchange"`
	Symbol               string                  `json:"symbol"`
	Interval             string                  `json:"interval"`
	StartTime            *time.Time              `json:"startTime,omitempty"`
	EndTime              *time.Time              `json:"endTime,omitempty"`
	RepairSourceTaskID   string                  `json:"repairSourceTaskId,omitempty"`
	RequestID            string                  `json:"requestId,omitempty"`
	SyncEnabled          bool                    `json:"syncEnabled"`
	RealtimeEnabled      bool                    `json:"realtimeEnabled"`
	Status               TaskStatus              `json:"status"`
	MarketStatus         DataSyncMarketStatus    `json:"marketStatus"`
	MarketStatusDetail   string                  `json:"marketStatusDetail,omitempty"`
	DataHealth           DataSyncHealth          `json:"dataHealth"`
	GapSummary           *DataSyncGapSummary     `json:"gapSummary,omitempty"`
	InvalidSummary       *DataSyncInvalidSummary `json:"invalidSummary,omitempty"`
	LatestSyncedOpenTime *time.Time              `json:"latestSyncedAt,omitempty"`
	LastError            string                  `json:"lastError,omitempty"`
	AttemptCount         int                     `json:"attemptCount"`
	NextAttemptAt        *time.Time              `json:"nextAttemptAt,omitempty"`
	ExchangeBackoffUntil *time.Time              `json:"exchangeBackoffUntil,omitempty"`
	ExchangeBackoffError string                  `json:"exchangeBackoffLastError,omitempty"`
	LockedBy             string                  `json:"lockedBy,omitempty"`
	LockedUntil          *time.Time              `json:"lockedUntil,omitempty"`
	HeartbeatAt          *time.Time              `json:"heartbeatAt,omitempty"`
	StartedAt            *time.Time              `json:"startedAt,omitempty"`
	FinishedAt           *time.Time              `json:"finishedAt,omitempty"`
	CreatedAt            time.Time               `json:"createdAt"`
	UpdatedAt            time.Time               `json:"updatedAt"`
}

type DataSyncGapSummary struct {
	Count    int        `json:"count"`
	FirstGap *CandleGap `json:"firstGap,omitempty"`
}

type DataSyncInvalidSummary struct {
	Count      int          `json:"count"`
	FirstIssue *CandleIssue `json:"firstIssue,omitempty"`
}

type DataSyncGapList struct {
	TaskID        string      `json:"taskId"`
	Gaps          []CandleGap `json:"gaps"`
	Limited       bool        `json:"limited"`
	TotalCount    int         `json:"totalCount"`
	ReturnedCount int         `json:"returnedCount"`
	RepairLimit   int         `json:"repairLimit"`
}

type DataSyncInvalidIssueList struct {
	TaskID        string        `json:"taskId"`
	Issues        []CandleIssue `json:"issues"`
	Limited       bool          `json:"limited"`
	TotalCount    int           `json:"totalCount"`
	ReturnedCount int           `json:"returnedCount"`
	IssueLimit    int           `json:"issueLimit"`
	Offset        int           `json:"offset"`
}

type DataSyncInvalidIssueQuery struct {
	Limit  int
	Offset int
	Code   string
	From   *time.Time
	To     *time.Time
}

func NormalizeDataSyncInvalidIssueQuery(query DataSyncInvalidIssueQuery) DataSyncInvalidIssueQuery {
	if query.Limit <= 0 {
		query.Limit = DefaultDataSyncInvalidIssueLimit
	}
	if query.Limit > MaxDataSyncInvalidIssueLimit {
		query.Limit = MaxDataSyncInvalidIssueLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func CandleIssueCodes() []string {
	return append([]string(nil), candleIssueCodes...)
}

func IsCandleIssueCode(value string) bool {
	for _, code := range candleIssueCodes {
		if value == code {
			return true
		}
	}
	return false
}

func IsRepairableCandleIssueCode(value string) bool {
	if value == CandleIssueInvalidOpenTime {
		return false
	}
	return IsCandleIssueCode(value)
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

type RepairDataSyncInvalidIssuesRequest struct {
	Code string     `json:"code,omitempty"`
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

type DataSyncResult struct {
	TaskID       string
	WorkerID     string
	Candles      []Candle
	LastOpenTime *time.Time
	Completed    bool
}

type CreateDataSyncTask struct {
	Exchange  string     `json:"exchange"`
	Symbol    string     `json:"symbol"`
	Interval  string     `json:"interval"`
	StartTime *time.Time `json:"startTime,omitempty"`
	EndTime   *time.Time `json:"endTime,omitempty"`
	RequestID string     `json:"-"`
}
