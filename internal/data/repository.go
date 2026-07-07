package data

import (
	"context"
	"time"
)

type Repository interface {
	ListDataSyncTasks(ctx context.Context) ([]DataSyncTask, error)
	GetDataSyncTask(ctx context.Context, id string) (DataSyncTask, error)
	CreateDataSyncTask(ctx context.Context, task CreateDataSyncTask) (DataSyncTask, error)
	DeleteDataSyncTask(ctx context.Context, id string) error
	RetryDataSyncTask(ctx context.Context, id string) (DataSyncTask, error)
	ListDataSyncTaskGaps(ctx context.Context, id string) (DataSyncGapList, error)
	ListDataSyncTaskInvalidIssues(ctx context.Context, id string, query DataSyncInvalidIssueQuery) (DataSyncInvalidIssueList, error)
	RepairDataSyncTaskGap(ctx context.Context, id string, request RepairDataSyncTaskGapRequest) (DataSyncGapRepairResult, error)
	RepairDataSyncTaskGaps(ctx context.Context, id string, request RepairDataSyncTaskGapsRequest) (DataSyncGapRepairResult, error)
	RepairDataSyncTaskInvalidIssues(ctx context.Context, id string, request RepairDataSyncInvalidIssuesRequest) (DataSyncGapRepairResult, error)
	SetSyncEnabled(ctx context.Context, id string, enabled bool) (DataSyncTask, error)
	SetRealtimeEnabled(ctx context.Context, id string, enabled bool) (DataSyncTask, error)
	GetCandles(ctx context.Context, query CandleQuery) (CandleResult, error)
	ListCandles(ctx context.Context, query CandleQuery) ([]Candle, error)
	ScanMarketCandleGaps(ctx context.Context, query MarketCandleGapScanQuery) (MarketCandleGapScan, error)
	ScanMarketCandleInvalidIssues(ctx context.Context, query MarketCandleInvalidIssueScanQuery) (MarketCandleInvalidIssueScan, error)
	RepairMarketCandleGap(ctx context.Context, request RepairMarketCandleGapRequest) (DataSyncGapRepairResult, error)
	RepairMarketCandleGaps(ctx context.Context, request RepairMarketCandleGapsRequest) (DataSyncGapRepairResult, error)
	RepairMarketCandleInvalidIssues(ctx context.Context, request RepairMarketCandleInvalidIssuesRequest) (DataSyncGapRepairResult, error)
	QuarantineMarketCandleInvalidIssues(ctx context.Context, request QuarantineMarketCandleInvalidIssuesRequest) (MarketCandleQuarantineResult, error)
	ListOverviewRecentFacts(ctx context.Context, query OverviewRecentFactQuery) (OverviewRecentFacts, error)
	ListOverviewTrends(ctx context.Context, query OverviewTrendQuery) (OverviewTrends, error)
	ListBacktestTasks(ctx context.Context) ([]BacktestTask, error)
	CreateBacktestTask(ctx context.Context, task CreateBacktestTask) (BacktestTask, error)
	GetBacktestTask(ctx context.Context, id string) (BacktestTask, error)
	ListBacktestIntents(ctx context.Context, backtestID string) ([]StrategyIntent, error)
	ListBacktestOrders(ctx context.Context, backtestID string) ([]BacktestOrder, error)
	ListTradingTasks(ctx context.Context) ([]TradingTask, error)
	CreateTradingTask(ctx context.Context, task CreateTradingTask) (TradingTask, error)
	GetTradingTask(ctx context.Context, id string) (TradingTask, error)
	SetTradingTaskStatus(ctx context.Context, id string, status TaskStatus) (TradingTask, error)
	ListTradingIntents(ctx context.Context, taskID string) ([]StrategyIntent, error)
	ListTradingOrders(ctx context.Context, taskID string) ([]Order, error)
	ListTradingExecutions(ctx context.Context, taskID string) ([]Execution, error)
	ListTradingPositions(ctx context.Context, taskID string) ([]Position, error)
	ListTradingNotifications(ctx context.Context, taskID string) ([]Notification, error)
	ListNotifications(ctx context.Context) ([]Notification, error)
	RetryNotification(ctx context.Context, id string) (Notification, error)
	ListNotificationChannels(ctx context.Context) ([]NotificationChannel, error)
	CreateNotificationChannel(ctx context.Context, channel CreateNotificationChannel) (NotificationChannel, error)
	UpdateNotificationChannel(ctx context.Context, id string, channel CreateNotificationChannel) (NotificationChannel, error)
	DeleteNotificationChannel(ctx context.Context, id string) (NotificationChannel, error)
	SetNotificationChannelEnabled(ctx context.Context, id string, enabled bool) (NotificationChannel, error)
	ListExchangeAccounts(ctx context.Context) ([]ExchangeAccount, error)
	GetExchangeAccount(ctx context.Context, exchange string, accountID string) (ExchangeAccount, error)
	GetActiveMarketInstrument(ctx context.Context, exchange string, symbol string) (MarketInstrument, error)
	CreateExchangeAccount(ctx context.Context, account CreateExchangeAccount) (ExchangeAccount, error)
	ListMarketInstruments(ctx context.Context, query MarketInstrumentQuery) ([]MarketInstrument, error)
	ListMarketInstrumentSyncStatuses(ctx context.Context) ([]MarketInstrumentSyncStatus, error)
	ReplaceMarketInstruments(ctx context.Context, exchange string, instruments []MarketInstrument, syncedAt time.Time) (MarketInstrumentSyncResult, error)
	RecordMarketInstrumentSyncFailure(ctx context.Context, exchange string, syncErr error, attemptedAt time.Time) error
	ListOperators(ctx context.Context) ([]Operator, error)
	CreateOperator(ctx context.Context, operator CreateOperator) (Operator, error)
	SetOperatorEnabled(ctx context.Context, id string, enabled bool) (Operator, error)
	SetOperatorRole(ctx context.Context, id string, role string) (OperatorRoleUpdateResult, error)
	AuthenticateOperator(ctx context.Context, username string, password string) (Operator, error)
	ChangeOperatorPassword(ctx context.Context, operatorID string, currentTokenHash string, currentPassword string, newPassword string) (int, error)
	CheckLoginRateLimit(ctx context.Context, keyHash string, now time.Time, window time.Duration) (bool, error)
	RecordLoginFailure(ctx context.Context, keyHash string, now time.Time, limit int, window time.Duration, lockout time.Duration) error
	ClearLoginRateLimit(ctx context.Context, keyHash string) error
	CreateOperatorSession(ctx context.Context, session OperatorSession) error
	GetOperatorBySession(ctx context.Context, tokenHash string, now time.Time) (Operator, error)
	DeleteOperatorSession(ctx context.Context, tokenHash string) error
	ListOperatorSessions(ctx context.Context, operatorID string, currentTokenHash string, now time.Time) ([]OperatorSession, error)
	DeleteOperatorSessionByID(ctx context.Context, operatorID string, sessionID string, currentTokenHash string) error
	RecordAuditEvent(ctx context.Context, event CreateAuditEvent) (AuditEvent, error)
	ListAuditEvents(ctx context.Context, limit int) ([]AuditEvent, error)
	ListAuditEventPage(ctx context.Context, query AuditEventListQuery) (AuditEventPage, error)
	VerifyAuditEventHashChain(ctx context.Context) (AuditEventHashChainVerification, error)
	SystemHealth(ctx context.Context) (SystemHealth, error)
}
