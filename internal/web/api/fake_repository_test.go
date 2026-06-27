package api

import (
	"context"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
)

type fakeRepository struct {
	backtestOrders  map[string][]data.BacktestOrder
	backtestIntents map[string][]data.StrategyIntent
	backtests       []data.BacktestTask
	channels        []data.NotificationChannel
	accounts        []data.ExchangeAccount
	operators       []data.Operator
	passwords       map[string]string
	sessions        map[string]data.OperatorSession
	tradingTasks    []data.TradingTask
	tasks           []data.DataSyncTask
	candles         []data.Candle
}

func newFakeRepository() *fakeRepository {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repository := &fakeRepository{
		backtestOrders:  map[string][]data.BacktestOrder{},
		backtestIntents: map[string][]data.StrategyIntent{},
		passwords:       map[string]string{},
		sessions:        map[string]data.OperatorSession{},
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
	return repository
}

func (repository *fakeRepository) ListDataSyncTasks(context.Context) ([]data.DataSyncTask, error) {
	return append([]data.DataSyncTask(nil), repository.tasks...), nil
}

func (repository *fakeRepository) CreateDataSyncTask(
	_ context.Context,
	request data.CreateDataSyncTask,
) (data.DataSyncTask, error) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	task := data.DataSyncTask{
		ID:        "dst_1",
		Exchange:  request.Exchange,
		Symbol:    request.Symbol,
		Interval:  request.Interval,
		Status:    data.TaskStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
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

func (repository *fakeRepository) SetSyncEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	return repository.updateTask(ctx, id, func(task *data.DataSyncTask) {
		task.SyncEnabled = enabled
		task.Status = data.TaskStatusPending
		if !enabled {
			task.Status = data.TaskStatusPaused
		}
	})
}

func (repository *fakeRepository) SetRealtimeEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) (data.DataSyncTask, error) {
	return repository.updateTask(ctx, id, func(task *data.DataSyncTask) {
		task.RealtimeEnabled = enabled
		task.Status = data.TaskStatusRunning
		if !enabled {
			task.Status = data.TaskStatusPaused
		}
	})
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

func (repository *fakeRepository) ListNativeCandles(
	_ context.Context,
	query data.CandleQuery,
) ([]data.Candle, error) {
	matches := make([]data.Candle, 0)
	for _, candle := range repository.candles {
		if candle.Exchange == query.Exchange && candle.Symbol == query.Symbol && candle.Interval == query.Interval {
			matches = append(matches, candle)
		}
	}
	return matches, nil
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

func (repository *fakeRepository) ListTradingIntents(context.Context, string) ([]data.StrategyIntent, error) {
	return nil, nil
}

func (repository *fakeRepository) ListTradingOrders(context.Context, string) ([]data.Order, error) {
	return nil, nil
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
		ID:        "ea_1",
		Exchange:  request.Exchange,
		Alias:     request.Alias,
		Enabled:   request.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	repository.accounts = append(repository.accounts, account)
	return account, nil
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

func (repository *fakeRepository) SystemHealth(context.Context) (data.SystemHealth, error) {
	return data.SystemHealth{
		Status:    "ok",
		Database:  "ok",
		CheckedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Services:  []data.ServiceHealth{{Name: "api", Status: "ok"}},
	}, nil
}

func (repository *fakeRepository) updateTask(
	_ context.Context,
	id string,
	update func(task *data.DataSyncTask),
) (data.DataSyncTask, error) {
	for index := range repository.tasks {
		if repository.tasks[index].ID == id {
			update(&repository.tasks[index])
			return repository.tasks[index], nil
		}
	}
	return data.DataSyncTask{}, data.ErrNotFound
}
