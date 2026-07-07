package api

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type fakeRepository struct {
	backtestOrders            map[string][]data.BacktestOrder
	backtestIntents           map[string][]data.StrategyIntent
	tradingOrders             map[string][]data.Order
	tradingIntents            map[string][]data.StrategyIntent
	backtests                 []data.BacktestTask
	channels                  []data.NotificationChannel
	notifications             []data.Notification
	accounts                  []data.ExchangeAccount
	auditEvents               []data.AuditEvent
	marketInstruments         []data.MarketInstrument
	marketSyncStatuses        []data.MarketInstrumentSyncStatus
	marketSyncFailures        []marketSyncFailure
	catalogPausedTasks        map[string]fakeCatalogPauseState
	operators                 []data.Operator
	passwords                 map[string]string
	loginRateLimits           map[string]data.LoginRateLimitState
	sessions                  map[string]data.OperatorSession
	tradingTasks              []data.TradingTask
	tasks                     []data.DataSyncTask
	taskGapDetails            map[string]data.DataSyncGapList
	taskInvalidDetails        map[string]data.DataSyncInvalidIssueList
	candles                   []data.Candle
	lastSystemHealthRequestID string
}

type marketSyncFailure struct {
	exchange    string
	err         error
	attemptedAt time.Time
}

type fakeCatalogPauseState struct {
	syncEnabled     bool
	realtimeEnabled bool
}

type failingListRepository struct {
	*fakeRepository
	err error
}

func (repository *failingListRepository) ListDataSyncTasks(context.Context) ([]data.DataSyncTask, error) {
	return nil, repository.err
}

func newFakeRepository() *fakeRepository {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		backtestOrders:     map[string][]data.BacktestOrder{},
		backtestIntents:    map[string][]data.StrategyIntent{},
		tradingOrders:      map[string][]data.Order{},
		tradingIntents:     map[string][]data.StrategyIntent{},
		catalogPausedTasks: map[string]fakeCatalogPauseState{},
		passwords:          map[string]string{},
		loginRateLimits:    map[string]data.LoginRateLimitState{},
		sessions:           map[string]data.OperatorSession{},
		taskGapDetails:     map[string]data.DataSyncGapList{},
		taskInvalidDetails: map[string]data.DataSyncInvalidIssueList{},
	}
	operator := data.Operator{
		ID:        "op_admin",
		Username:  testUsername,
		Enabled:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.operators = append(repository.operators, operator)
	repository.passwords[operator.ID] = testPassword
	repository.marketInstruments = defaultFakeMarketInstruments(now)
	repository.marketSyncStatuses = []data.MarketInstrumentSyncStatus{{
		Exchange:      "binance",
		LastAttemptAt: now,
		LastSuccessAt: &now,
		UpdatedAt:     now,
	}}
	return repository
}

func (repository *fakeRepository) ListDataSyncTasks(context.Context) ([]data.DataSyncTask, error) {
	tasks := make([]data.DataSyncTask, len(repository.tasks))
	for index, task := range repository.tasks {
		tasks[index] = normalizeFakeDataSyncTask(task)
	}
	return tasks, nil
}

func (repository *fakeRepository) GetDataSyncTask(_ context.Context, id string) (data.DataSyncTask, error) {
	for _, task := range repository.tasks {
		if task.ID == id {
			return normalizeFakeDataSyncTask(task), nil
		}
	}
	return data.DataSyncTask{}, data.ErrNotFound
}

func (repository *fakeRepository) CreateDataSyncTask(
	_ context.Context,
	request data.CreateDataSyncTask,
) (data.DataSyncTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.DataSyncTask{
		ID:           "dst_1",
		Exchange:     request.Exchange,
		Symbol:       request.Symbol,
		Interval:     request.Interval,
		RequestID:    request.RequestID,
		TraceParent:  request.TraceParent,
		Status:       data.TaskStatusPending,
		MarketStatus: data.DataSyncMarketStatusActive,
		DataHealth:   data.DataSyncHealthInsufficient,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	repository.tasks = append(repository.tasks, task)
	return task, nil
}

func (repository *fakeRepository) DeleteDataSyncTask(_ context.Context, id string) error {
	for index, task := range repository.tasks {
		if task.ID == id {
			repository.tasks = append(repository.tasks[:index], repository.tasks[index+1:]...)
			return nil
		}
	}
	return data.ErrNotFound
}

func (repository *fakeRepository) RetryDataSyncTask(
	_ context.Context,
	id string,
) (data.DataSyncTask, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID == id {
			if repository.tasks[index].Status != data.TaskStatusFailed {
				return data.DataSyncTask{}, data.DataSyncRetryRequiresFailedError()
			}
			if normalizeFakeDataSyncTask(repository.tasks[index]).MarketStatus != data.DataSyncMarketStatusActive {
				return data.DataSyncTask{}, data.MarketInstrumentNotActiveError()
			}
			repository.tasks[index].SyncEnabled = true
			repository.tasks[index].Status = data.TaskStatusPending
			repository.tasks[index].LastError = ""
			refreshFakeDataSyncHealth(&repository.tasks[index])
			return repository.tasks[index], nil
		}
	}
	return data.DataSyncTask{}, data.ErrNotFound
}

func (repository *fakeRepository) ListDataSyncTaskGaps(
	_ context.Context,
	id string,
) (data.DataSyncGapList, error) {
	for _, task := range repository.tasks {
		if task.ID != id {
			continue
		}
		if detail, ok := repository.taskGapDetails[id]; ok {
			return cloneDataSyncGapList(detail), nil
		}
		result := data.DataSyncGapList{
			TaskID:      task.ID,
			Gaps:        []data.CandleGap{},
			RepairLimit: 20,
		}
		if task.GapSummary != nil && task.GapSummary.FirstGap != nil && task.GapSummary.Count > 0 {
			result.Gaps = append(result.Gaps, *task.GapSummary.FirstGap)
			result.TotalCount = task.GapSummary.Count
			result.ReturnedCount = len(result.Gaps)
			result.Limited = task.GapSummary.Count > result.ReturnedCount
		}
		return result, nil
	}
	return data.DataSyncGapList{}, data.ErrNotFound
}

func (repository *fakeRepository) GetCandles(
	_ context.Context,
	query data.CandleQuery,
) (data.CandleResult, error) {
	return data.NewCandleProvider(repository).GetCandles(context.Background(), query)
}

func (repository *fakeRepository) ListCandles(
	ctx context.Context,
	query data.CandleQuery,
) ([]data.Candle, error) {
	result, err := repository.GetCandles(ctx, query)
	if err != nil {
		return nil, err
	}
	return result.Candles, nil
}

func (repository *fakeRepository) ListBacktestTasks(context.Context) ([]data.BacktestTask, error) {
	return append([]data.BacktestTask(nil), repository.backtests...), nil
}

func (repository *fakeRepository) CreateBacktestTask(
	_ context.Context,
	request data.CreateBacktestTask,
) (data.BacktestTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.BacktestTask{
		ID:             "bt_1",
		Name:           request.Name,
		Exchange:       request.Exchange,
		Symbol:         request.Symbol,
		Interval:       request.Interval,
		StartTime:      request.StartTime,
		EndTime:        request.EndTime,
		RequestID:      request.RequestID,
		TraceParent:    request.TraceParent,
		StrategyID:     request.StrategyID,
		StrategyParams: request.StrategyParams,
		InitialBalance: request.InitialBalance,
		FeeBps:         request.FeeBps,
		SlippageBps:    request.SlippageBps,
		TriggerMode:    request.TriggerMode,
		Status:         data.TaskStatusPending,
		ResultSummary:  map[string]any{},
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repository.backtests = append(repository.backtests, task)
	return task, nil
}

func (repository *fakeRepository) GetBacktestTask(_ context.Context, id string) (data.BacktestTask, error) {
	for _, task := range repository.backtests {
		if task.ID == id {
			return task, nil
		}
	}
	return data.BacktestTask{}, data.ErrNotFound
}

func (repository *fakeRepository) ListBacktestOrders(
	_ context.Context,
	backtestID string,
) ([]data.BacktestOrder, error) {
	return append([]data.BacktestOrder(nil), repository.backtestOrders[backtestID]...), nil
}

func (repository *fakeRepository) ListBacktestIntents(
	_ context.Context,
	backtestID string,
) ([]data.StrategyIntent, error) {
	return append([]data.StrategyIntent(nil), repository.backtestIntents[backtestID]...), nil
}

func (repository *fakeRepository) ListTradingTasks(context.Context) ([]data.TradingTask, error) {
	return append([]data.TradingTask(nil), repository.tradingTasks...), nil
}

func (repository *fakeRepository) CreateTradingTask(
	_ context.Context,
	request data.CreateTradingTask,
) (data.TradingTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.TradingTask{
		ID:             "tt_" + request.Type,
		Name:           request.Name,
		Type:           request.Type,
		Exchange:       request.Exchange,
		AccountID:      request.AccountID,
		Symbol:         request.Symbol,
		Interval:       request.Interval,
		StrategyID:     request.StrategyID,
		StrategyParams: request.StrategyParams,
		IntentPolicy:   request.IntentPolicy,
		RequestID:      request.RequestID,
		TraceParent:    request.TraceParent,
		Status:         data.TaskStatusPending,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	repository.tradingTasks = append(repository.tradingTasks, task)
	return task, nil
}

func (repository *fakeRepository) GetTradingTask(_ context.Context, id string) (data.TradingTask, error) {
	for _, task := range repository.tradingTasks {
		if task.ID == id {
			return task, nil
		}
	}
	return data.TradingTask{}, data.ErrNotFound
}

func (repository *fakeRepository) SetTradingTaskStatus(
	_ context.Context,
	id string,
	status data.TaskStatus,
) (data.TradingTask, error) {
	for index := range repository.tradingTasks {
		if repository.tradingTasks[index].ID == id {
			repository.tradingTasks[index].Status = status
			return repository.tradingTasks[index], nil
		}
	}
	return data.TradingTask{}, data.ErrNotFound
}

func (repository *fakeRepository) ListTradingIntents(_ context.Context, taskID string) ([]data.StrategyIntent, error) {
	return append([]data.StrategyIntent(nil), repository.tradingIntents[taskID]...), nil
}

func (repository *fakeRepository) ListTradingOrders(_ context.Context, taskID string) ([]data.Order, error) {
	return append([]data.Order(nil), repository.tradingOrders[taskID]...), nil
}

func (repository *fakeRepository) ListTradingExecutions(context.Context, string) ([]data.Execution, error) {
	return nil, nil
}

func (repository *fakeRepository) ListTradingPositions(context.Context, string) ([]data.Position, error) {
	return nil, nil
}

func (repository *fakeRepository) ListTradingNotifications(context.Context, string) ([]data.Notification, error) {
	return nil, nil
}

func (repository *fakeRepository) ListNotifications(context.Context) ([]data.Notification, error) {
	return append([]data.Notification(nil), repository.notifications...), nil
}

func (repository *fakeRepository) RetryNotification(_ context.Context, id string) (data.Notification, error) {
	for index := range repository.notifications {
		if repository.notifications[index].ID == id {
			repository.notifications[index].Status = "pending"
			repository.notifications[index].Error = ""
			return repository.notifications[index], nil
		}
	}
	return data.Notification{}, data.ErrNotFound
}

func (repository *fakeRepository) ListNotificationChannels(context.Context) ([]data.NotificationChannel, error) {
	return append([]data.NotificationChannel(nil), repository.channels...), nil
}

func (repository *fakeRepository) CreateNotificationChannel(
	_ context.Context,
	request data.CreateNotificationChannel,
) (data.NotificationChannel, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	channel := data.NotificationChannel{
		ID:        "nc_1",
		Name:      request.Name,
		Provider:  request.Provider,
		Target:    request.Target,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.channels = append(repository.channels, channel)
	return channel, nil
}

func (repository *fakeRepository) ListExchangeAccounts(context.Context) ([]data.ExchangeAccount, error) {
	return append([]data.ExchangeAccount(nil), repository.accounts...), nil
}

func (repository *fakeRepository) CreateExchangeAccount(
	_ context.Context,
	request data.CreateExchangeAccount,
) (data.ExchangeAccount, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	account := data.ExchangeAccount{
		ID:               "ea_1",
		Exchange:         request.Exchange,
		Alias:            request.Alias,
		Enabled:          request.Enabled,
		CredentialStatus: "encrypted",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	repository.accounts = append(repository.accounts, account)
	return account, nil
}

func (repository *fakeRepository) GetExchangeAccount(
	_ context.Context,
	exchange string,
	accountID string,
) (data.ExchangeAccount, error) {
	for _, account := range repository.accounts {
		if account.Exchange == exchange && (account.ID == accountID || account.Alias == accountID) {
			return account, nil
		}
	}
	return data.ExchangeAccount{}, data.ErrNotFound
}

func (repository *fakeRepository) ListOperators(context.Context) ([]data.Operator, error) {
	return append([]data.Operator(nil), repository.operators...), nil
}

func (repository *fakeRepository) CreateOperator(
	_ context.Context,
	request data.CreateOperator,
) (data.Operator, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	operator := data.Operator{
		ID:        "op_" + request.Username,
		Username:  request.Username,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.operators = append(repository.operators, operator)
	repository.passwords[operator.ID] = request.Password
	return operator, nil
}

func (repository *fakeRepository) SetOperatorEnabled(
	_ context.Context,
	id string,
	enabled bool,
) (data.Operator, error) {
	for index := range repository.operators {
		if repository.operators[index].ID == id {
			repository.operators[index].Enabled = enabled
			return repository.operators[index], nil
		}
	}
	return data.Operator{}, data.ErrNotFound
}

func (repository *fakeRepository) AuthenticateOperator(
	_ context.Context,
	username string,
	password string,
) (data.Operator, error) {
	for _, operator := range repository.operators {
		if operator.Username == username && operator.Enabled && repository.passwords[operator.ID] == password {
			return operator, nil
		}
	}
	return data.Operator{}, data.ErrUnauthorized
}

func (repository *fakeRepository) CreateOperatorSession(
	_ context.Context,
	session data.OperatorSession,
) error {
	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	}
	repository.sessions[session.TokenHash] = session
	return nil
}

func (repository *fakeRepository) GetOperatorBySession(
	_ context.Context,
	tokenHash string,
	now time.Time,
) (data.Operator, error) {
	session, exists := repository.sessions[tokenHash]
	if !exists || !session.ExpiresAt.After(now) {
		return data.Operator{}, data.ErrUnauthorized
	}
	for _, operator := range repository.operators {
		if operator.ID == session.OperatorID && operator.Enabled {
			return operator, nil
		}
	}
	return data.Operator{}, data.ErrUnauthorized
}

func (repository *fakeRepository) DeleteOperatorSession(_ context.Context, tokenHash string) error {
	delete(repository.sessions, tokenHash)
	return nil
}

func (repository *fakeRepository) ListOperatorSessions(
	_ context.Context,
	operatorID string,
	currentTokenHash string,
	now time.Time,
) ([]data.OperatorSession, error) {
	sessions := []data.OperatorSession{}
	for tokenHash, session := range repository.sessions {
		if session.OperatorID != operatorID || !session.ExpiresAt.After(now) {
			continue
		}
		session.TokenHash = tokenHash
		session.Current = tokenHash == currentTokenHash
		sessions = append(sessions, session)
	}
	sort.SliceStable(sessions, func(left int, right int) bool {
		if sessions[left].Current != sessions[right].Current {
			return sessions[left].Current
		}
		return sessions[left].CreatedAt.After(sessions[right].CreatedAt)
	})
	return sessions, nil
}

func (repository *fakeRepository) DeleteOperatorSessionByID(
	_ context.Context,
	operatorID string,
	sessionID string,
	currentTokenHash string,
) error {
	for tokenHash, session := range repository.sessions {
		if session.ID != sessionID || session.OperatorID != operatorID {
			continue
		}
		if tokenHash == currentTokenHash {
			return data.ErrInvalidState
		}
		delete(repository.sessions, tokenHash)
		return nil
	}
	return data.ErrNotFound
}

func (repository *fakeRepository) RecordAuditEvent(
	_ context.Context,
	request data.CreateAuditEvent,
) (data.AuditEvent, error) {
	event := data.AuditEvent{
		ID:              "ae_" + strconv.Itoa(len(repository.auditEvents)+1),
		ActorOperatorID: request.ActorOperatorID,
		ActorUsername:   request.ActorUsername,
		Action:          request.Action,
		ResourceType:    request.ResourceType,
		ResourceID:      request.ResourceID,
		Outcome:         request.Outcome,
		RequestMethod:   request.RequestMethod,
		RequestPath:     request.RequestPath,
		RemoteAddr:      request.RemoteAddr,
		UserAgent:       request.UserAgent,
		Metadata:        request.Metadata,
		CreatedAt:       time.Date(2026, 1, 1, 0, 0, len(repository.auditEvents), 0, time.UTC),
	}
	if event.Metadata == nil {
		event.Metadata = map[string]string{}
	}
	repository.auditEvents = append(repository.auditEvents, event)
	return event, nil
}

func (repository *fakeRepository) ListAuditEvents(_ context.Context, limit int) ([]data.AuditEvent, error) {
	events := append([]data.AuditEvent(nil), repository.auditEvents...)
	sort.SliceStable(events, func(left int, right int) bool {
		return events[left].CreatedAt.After(events[right].CreatedAt)
	})
	if limit <= 0 || limit > len(events) {
		return events, nil
	}
	return events[:limit], nil
}

func (repository *fakeRepository) SystemHealth(ctx context.Context) (data.SystemHealth, error) {
	repository.lastSystemHealthRequestID = RequestIDFromContext(ctx)
	pendingCount := 1
	runningCount := 1
	lockedCount := 1
	staleLeaseCount := 0
	heartbeat := time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC)
	lockedUntil := time.Date(2026, 1, 1, 0, 2, 0, 0, time.UTC)
	return data.SystemHealth{
		Status:    "ok",
		Database:  "ok",
		CheckedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Services: []data.ServiceHealth{
			{Name: "api", Status: "ok"},
			{
				Name:            "trading-worker",
				Status:          "ok",
				Detail:          "pending=1 running=1 locked=1 stale=0",
				PendingCount:    &pendingCount,
				RunningCount:    &runningCount,
				LockedCount:     &lockedCount,
				StaleLeaseCount: &staleLeaseCount,
				LastHeartbeatAt: &heartbeat,
				LockedUntil:     &lockedUntil,
			},
		},
	}, nil
}

func (repository *fakeRepository) fakeRepairTaskExists(source data.DataSyncTask, gap data.CandleGap) bool {
	for _, task := range repository.tasks {
		if task.Exchange == source.Exchange &&
			task.Symbol == source.Symbol &&
			task.Interval == source.Interval &&
			task.StartTime != nil &&
			task.EndTime != nil &&
			task.StartTime.Equal(gap.From) &&
			task.EndTime.Equal(gap.To) {
			return true
		}
	}
	return false
}

func cloneDataSyncGapList(value data.DataSyncGapList) data.DataSyncGapList {
	return data.DataSyncGapList{
		TaskID:        value.TaskID,
		Gaps:          append([]data.CandleGap(nil), value.Gaps...),
		Limited:       value.Limited,
		TotalCount:    value.TotalCount,
		ReturnedCount: value.ReturnedCount,
		RepairLimit:   value.RepairLimit,
	}
}

func dataSyncTaskCommandAllowed(status data.TaskStatus) bool {
	return status == data.TaskStatusPending ||
		status == data.TaskStatusRunning ||
		status == data.TaskStatusPaused ||
		status == data.TaskStatusSucceeded
}

func refreshFakeDataSyncHealth(task *data.DataSyncTask) {
	if task.Status == data.TaskStatusFailed || task.Status == data.TaskStatusCancelled {
		task.DataHealth = data.DataSyncHealthFailed
		return
	}
	if task.NextAttemptAt != nil && task.NextAttemptAt.After(time.Now().UTC()) {
		task.DataHealth = data.DataSyncHealthRetrying
		return
	}
	if task.Status == data.TaskStatusPaused {
		task.DataHealth = data.DataSyncHealthPaused
		return
	}
	if task.LatestSyncedOpenTime == nil {
		if (task.Status == data.TaskStatusPending || task.Status == data.TaskStatusRunning) &&
			(task.SyncEnabled || task.RealtimeEnabled) {
			task.DataHealth = data.DataSyncHealthSyncing
			return
		}
		task.DataHealth = data.DataSyncHealthInsufficient
		return
	}
	if (task.Status == data.TaskStatusPending || task.Status == data.TaskStatusRunning) &&
		(task.SyncEnabled || task.RealtimeEnabled) {
		task.DataHealth = data.DataSyncHealthSyncing
		return
	}
	task.DataHealth = data.DataSyncHealthOK
}
