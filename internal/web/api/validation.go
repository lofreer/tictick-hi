package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

func validateCreateTask(task data.CreateDataSyncTask) error {
	if task.Exchange == "" || task.Symbol == "" || task.Interval == "" {
		return errors.New("exchange, symbol and interval are required")
	}
	return nil
}

func validateNotificationChannel(channel data.CreateNotificationChannel) error {
	if channel.Name == "" || channel.Provider == "" || channel.Target == "" {
		return errors.New("name, provider and target are required")
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
	if len(operator.Password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	return nil
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
	return validateStrategyParams(definition, task.StrategyParams)
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
		confirmed, ok := task.IntentPolicy["liveExecutionConfirmed"].(bool)
		if !ok || !confirmed {
			return errors.New("live execution must be explicitly confirmed")
		}
	}
	return validateStrategyParams(definition, task.StrategyParams)
}

func validateStrategyParams(definition strategy.Definition, values map[string]any) error {
	for _, param := range definition.Params {
		value, exists := values[param.Key]
		if !exists {
			if param.Required {
				return fmt.Errorf("%s is required", param.Key)
			}
			continue
		}
		if !validStrategyParamValue(param, value) {
			return fmt.Errorf("%s has invalid value", param.Key)
		}
	}
	return nil
}

func validStrategyParamValue(param strategy.ParamSpec, value any) bool {
	switch param.Type {
	case "number":
		return numberValue(value)
	case "select":
		text, ok := value.(string)
		if !ok || text == "" {
			return false
		}
		if len(param.Options) == 0 {
			return true
		}
		for _, option := range param.Options {
			if option.Value == text {
				return true
			}
		}
		return false
	case "boolean":
		_, ok := value.(bool)
		return ok
	default:
		text, ok := value.(string)
		return ok && text != ""
	}
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

func numberValue(value any) bool {
	switch typed := value.(type) {
	case float64:
		return true
	case float32:
		return true
	case int:
		return true
	case int64:
		return true
	case json.Number:
		_, err := typed.Float64()
		return err == nil
	default:
		return false
	}
}
