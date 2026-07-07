package api

import (
	"encoding/json"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/notification"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

var (
	binanceSymbolPattern = regexp.MustCompile(`^[A-Z0-9]{3,30}$`)
	okxSymbolPattern     = regexp.MustCompile(`^[A-Z0-9]+-[A-Z0-9]+(?:-[A-Z0-9]+)?$`)
)

func validateCreateTask(task data.CreateDataSyncTask) error {
	if strings.TrimSpace(task.Exchange) == "" ||
		strings.TrimSpace(task.Symbol) == "" ||
		strings.TrimSpace(task.Interval) == "" {
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
	if strings.TrimSpace(channel.Name) == "" || channel.Provider == "" || strings.TrimSpace(channel.Target) == "" {
		return errors.New("name, provider and target are required")
	}
	switch channel.Provider {
	case "local", "webhook-demo", "webhook", "email", "telegram", "feishu":
	default:
		return errors.New("provider must be local, webhook-demo, webhook, email, telegram or feishu")
	}
	return notification.ValidateProviderTargetSyntax(channel.Provider, channel.Target)
}

func validateExchangeAccount(account data.CreateExchangeAccount) error {
	if strings.TrimSpace(account.Exchange) == "" ||
		strings.TrimSpace(account.Alias) == "" ||
		strings.TrimSpace(account.APIKey) == "" ||
		strings.TrimSpace(account.APISecret) == "" {
		return errors.New("exchange, alias, apiKey and apiSecret are required")
	}
	return nil
}

func validateOperator(operator data.CreateOperator) error {
	if strings.TrimSpace(operator.Username) == "" || strings.TrimSpace(operator.Password) == "" {
		return errors.New("username and password are required")
	}
	role := data.NormalizeCreateOperatorRole(operator.Role)
	if err := data.ValidateOperatorRole(role); err != nil {
		return err
	}
	return data.ValidateOperatorPasswordForUsername(operator.Username, operator.Password)
}

func validateOperatorRoleUpdate(request data.UpdateOperatorRole) error {
	return data.ValidateOperatorRole(data.NormalizeOperatorRole(request.Role))
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
	if strings.TrimSpace(task.Name) == "" ||
		strings.TrimSpace(task.Exchange) == "" ||
		strings.TrimSpace(task.Symbol) == "" ||
		strings.TrimSpace(task.Interval) == "" ||
		strings.TrimSpace(task.StrategyID) == "" {
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
	if !validBasisPoints(task.FeeBps) || !validBasisPoints(task.SlippageBps) {
		return errors.New("feeBps and slippageBps must be numbers between 0 and 10000")
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
	if _, exists := task.IntentPolicy["riskLimitPct"]; !exists {
		task.IntentPolicy["riskLimitPct"] = 10.0
	}
}

func validateCreateTradingTask(task data.CreateTradingTask, definition strategy.Definition) error {
	if strings.TrimSpace(task.Name) == "" ||
		strings.TrimSpace(task.Type) == "" ||
		strings.TrimSpace(task.Exchange) == "" ||
		strings.TrimSpace(task.Symbol) == "" ||
		strings.TrimSpace(task.Interval) == "" ||
		strings.TrimSpace(task.StrategyID) == "" {
		return errors.New("name, type, exchange, symbol, interval and strategyId are required")
	}
	if err := validateExchangeSymbol(task.Exchange, task.Symbol); err != nil {
		return err
	}
	if task.Type != "paper" && task.Type != "live" {
		return errors.New("type must be paper or live")
	}
	if task.Type == "live" && strings.TrimSpace(task.LiveConfirmation) != "LIVE" {
		return errors.New("liveConfirmation must be LIVE for live tasks")
	}
	if !contains(definition.SupportedIntervals, task.Interval) {
		return errors.New("strategy does not support interval")
	}
	if strings.TrimSpace(task.AccountID) == "" {
		return errors.New("accountId is required")
	}
	orderIntent, ok := task.IntentPolicy["orderIntent"].(string)
	if !ok || (orderIntent != "execute" && orderIntent != "notify") {
		return errors.New("intentPolicy.orderIntent must be execute or notify")
	}
	if task.Type == "live" && orderIntent == "execute" {
		return errors.New("live execution is disabled until the live safety stage")
	}
	if !validPolicyPercentage(task.IntentPolicy["riskLimitPct"]) {
		return errors.New("intentPolicy.riskLimitPct must be a number between 0 and 100")
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

func validBasisPoints(value string) bool {
	number, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return false
	}
	return number >= 0 && number <= 10000
}

func validPolicyPercentage(value any) bool {
	number, ok := policyNumber(value)
	return ok && number >= 0 && number <= 100
}

func policyNumber(value any) (float64, bool) {
	switch number := value.(type) {
	case float64:
		return number, true
	case float32:
		return float64(number), true
	case int:
		return float64(number), true
	case int64:
		return float64(number), true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
