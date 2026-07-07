package api

import (
	"errors"
	"regexp"
	"strconv"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

var (
	binanceSymbolPattern = regexp.MustCompile(`^[A-Z0-9]{3,30}$`)
	okxSymbolPattern     = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+(?:-[A-Z0-9]+)?$`)
)

func validateCreateTask(task data.CreateDataSyncTask) error {
	if task.Exchange == "" || task.Symbol == "" || task.Interval == "" {
		return errors.New("exchange, symbol and interval are required")
	}
	if err := validateExchangeSymbol(task.Exchange, task.Symbol); err != nil {
		return err
	}
	if err := data.ValidateDataSyncTaskWindow(task.Interval, task.StartTime, task.EndTime); err != nil {
		return err
	}
	return nil
}

func validateExchangeSymbol(exchange string, symbol string) error {
	switch exchange {
	case "binance":
		if !binanceSymbolPattern.MatchString(symbol) {
			return errors.New("binance symbol must use uppercase compact format such as BTCUSDT")
		}
	case "okx":
		if !okxSymbolPattern.MatchString(symbol) {
			return errors.New("okx symbol must use uppercase instrument format such as BTC-USDT")
		}
	default:
		return errors.New("exchange must be binance or okx")
	}
	return nil
}

func validateNotificationChannel(channel data.CreateNotificationChannel) error {
	if channel.Name == "" || channel.Provider == "" || channel.Target == "" {
		return errors.New("name, provider and target are required")
	}
	switch channel.Provider {
	case "local", "webhook-demo", "webhook", "email", "telegram", "feishu":
	default:
		return errors.New("provider must be local, webhook-demo, webhook, email, telegram or feishu")
	}
	return nil
}

func validateExchangeAccount(account data.CreateExchangeAccount) error {
	if account.Exchange == "" || account.Alias == "" || account.APIKey == "" || account.APISecret == "" {
		return errors.New("exchange, alias, apiKey and apiSecret are required")
	}
	return nil
}

func validateOperator(operator data.CreateOperator) error {
	if operator.Username == "" || operator.Password == "" {
		return errors.New("username and password are required")
	}
	return data.ValidateOperatorPassword(operator.Password)
}

func normalizeCreateBacktest(task *data.CreateBacktestTask) {
	if task.StrategyParams == nil {
		task.StrategyParams = map[string]any{}
	}
	if task.InitialBalance == "" {
		task.InitialBalance = "10000"
	}
	if task.FeeBps == "" {
		task.FeeBps = "0"
	}
	if task.SlippageBps == "" {
		task.SlippageBps = "0"
	}
	if task.TriggerMode == "" {
		task.TriggerMode = "closed_candle"
	}
}

func validateCreateBacktest(task data.CreateBacktestTask, definition strategy.Definition) error {
	if task.Name == "" || task.Exchange == "" || task.Symbol == "" || task.Interval == "" || task.StrategyID == "" {
		return errors.New("name, exchange, symbol, interval and strategyId are required")
	}
	if err := validateExchangeSymbol(task.Exchange, task.Symbol); err != nil {
		return err
	}
	if task.StartTime == nil || task.EndTime == nil {
		return errors.New("startTime and endTime are required")
	}
	if !task.StartTime.Before(*task.EndTime) {
		return errors.New("startTime must be before endTime")
	}
	if !contains(definition.SupportedIntervals, task.Interval) {
		return errors.New("strategy does not support interval")
	}
	if !validDecimal(task.InitialBalance, true) {
		return errors.New("initialBalance must be a positive number")
	}
	if !validDecimal(task.FeeBps, false) || !validDecimal(task.SlippageBps, false) {
		return errors.New("feeBps and slippageBps must be non-negative numbers")
	}
	if task.TriggerMode != "closed_candle" && task.TriggerMode != "minute_replay" {
		return errors.New("triggerMode must be closed_candle or minute_replay")
	}
	return strategy.ValidateParams(definition, task.StrategyParams)
}

func normalizeCreateTradingTask(task *data.CreateTradingTask) {
	if task.StrategyParams == nil {
		task.StrategyParams = map[string]any{}
	}
	if task.IntentPolicy == nil {
		task.IntentPolicy = map[string]any{}
	}
	if task.Type == "paper" && task.AccountID == "" {
		task.AccountID = "paper"
	}
	if _, exists := task.IntentPolicy["orderIntent"]; !exists {
		if task.Type == "live" {
			task.IntentPolicy["orderIntent"] = "notify"
		} else {
			task.IntentPolicy["orderIntent"] = "execute"
		}
	}
	if _, exists := task.IntentPolicy["notificationChannel"]; !exists {
		task.IntentPolicy["notificationChannel"] = "default"
	}
}

func validateCreateTradingTask(task data.CreateTradingTask, definition strategy.Definition) error {
	if task.Name == "" || task.Type == "" || task.Exchange == "" || task.Symbol == "" || task.Interval == "" || task.StrategyID == "" {
		return errors.New("name, type, exchange, symbol, interval and strategyId are required")
	}
	if err := validateExchangeSymbol(task.Exchange, task.Symbol); err != nil {
		return err
	}
	if task.Type != "paper" && task.Type != "live" {
		return errors.New("type must be paper or live")
	}
	if !contains(definition.SupportedIntervals, task.Interval) {
		return errors.New("strategy does not support interval")
	}
	if task.AccountID == "" {
		return errors.New("accountId is required")
	}
	orderIntent, ok := task.IntentPolicy["orderIntent"].(string)
	if !ok || (orderIntent != "execute" && orderIntent != "notify") {
		return errors.New("intentPolicy.orderIntent must be execute or notify")
	}
	if task.Type == "live" && orderIntent == "execute" {
		return errors.New("live execution is disabled until the live safety stage")
	}
	return strategy.ValidateParams(definition, task.StrategyParams)
}

func validDecimal(value string, positive bool) bool {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return false
	}
	if positive {
		return number > 0
	}
	return number >= 0
}
