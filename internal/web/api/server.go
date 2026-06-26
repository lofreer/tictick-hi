package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

type Server struct {
	repository         data.Repository
	strategyRepository strategy.Repository
	staticRoot         string
}

func NewServer(repository data.Repository, staticRoot string) http.Handler {
	return &Server{
		repository:         repository,
		strategyRepository: strategy.BuiltinRegistry(),
		staticRoot:         staticRoot,
	}
}

func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/readyz":
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case strings.HasPrefix(r.URL.Path, "/api/data/tasks"):
		server.handleDataTasks(w, r)
	case r.URL.Path == "/api/candles":
		server.handleCandles(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/strategies"):
		server.handleStrategies(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/backtests"):
		server.handleBacktests(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/trading/tasks"):
		server.handleTradingTasks(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/"):
		writeError(w, http.StatusNotFound, "api route not found")
	default:
		server.serveFrontend(w, r)
	}
}

func (server *Server) handleDataTasks(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 {
		server.handleTaskCollection(w, r)
		return
	}
	if len(parts) == 4 && r.Method == http.MethodDelete {
		server.deleteDataTask(w, r, parts[3])
		return
	}
	if len(parts) == 6 && r.Method == http.MethodPost {
		server.handleTaskCommand(w, r, taskCommand{
			id:       parts[3],
			category: parts[4],
			action:   parts[5],
		})
		return
	}
	writeError(w, http.StatusNotFound, "data task route not found")
}

func (server *Server) handleTaskCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := server.repository.ListDataSyncTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var request data.CreateDataSyncTask
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := validateCreateTask(request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task, err := server.repository.CreateDataSyncTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) handleTaskCommand(w http.ResponseWriter, r *http.Request, command taskCommand) {
	var (
		task data.DataSyncTask
		err  error
	)

	switch command {
	case taskCommand{id: command.id, category: "sync", action: "start"}:
		task, err = server.repository.SetSyncEnabled(r.Context(), command.id, true)
	case taskCommand{id: command.id, category: "sync", action: "stop"}:
		task, err = server.repository.SetSyncEnabled(r.Context(), command.id, false)
	case taskCommand{id: command.id, category: "realtime", action: "start"}:
		task, err = server.repository.SetRealtimeEnabled(r.Context(), command.id, true)
	case taskCommand{id: command.id, category: "realtime", action: "stop"}:
		task, err = server.repository.SetRealtimeEnabled(r.Context(), command.id, false)
	default:
		writeError(w, http.StatusNotFound, "data task command not found")
		return
	}

	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) deleteDataTask(w http.ResponseWriter, r *http.Request, id string) {
	if err := server.repository.DeleteDataSyncTask(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (server *Server) handleCandles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query, err := parseCandleQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	candles, err := server.repository.ListCandles(r.Context(), query)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, candles)
}

func (server *Server) handleBacktests(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 2 {
		server.handleBacktestCollection(w, r)
		return
	}
	if len(parts) == 3 && r.Method == http.MethodGet {
		server.getBacktest(w, r, parts[2])
		return
	}
	if len(parts) == 4 && parts[3] == "orders" && r.Method == http.MethodGet {
		server.listBacktestOrders(w, r, parts[2])
		return
	}
	writeError(w, http.StatusNotFound, "backtest route not found")
}

func (server *Server) handleBacktestCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := server.repository.ListBacktestTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var request data.CreateBacktestTask
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		normalizeCreateBacktest(&request)
		definition, err := server.strategyRepository.GetStrategy(r.Context(), request.StrategyID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := validateCreateBacktest(request, definition); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task, err := server.repository.CreateBacktestTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) getBacktest(w http.ResponseWriter, r *http.Request, id string) {
	task, err := server.repository.GetBacktestTask(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) listBacktestOrders(w http.ResponseWriter, r *http.Request, id string) {
	if _, err := server.repository.GetBacktestTask(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	orders, err := server.repository.ListBacktestOrders(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (server *Server) handleTradingTasks(w http.ResponseWriter, r *http.Request) {
	parts := pathParts(r.URL.Path)
	if len(parts) == 3 {
		server.handleTradingTaskCollection(w, r)
		return
	}
	if len(parts) == 4 && r.Method == http.MethodGet {
		server.getTradingTask(w, r, parts[3])
		return
	}
	if len(parts) == 5 && r.Method == http.MethodPost {
		server.handleTradingTaskCommand(w, r, parts[3], parts[4])
		return
	}
	if len(parts) == 5 && r.Method == http.MethodGet {
		server.handleTradingTaskDetailCollection(w, r, parts[3], parts[4])
		return
	}
	writeError(w, http.StatusNotFound, "trading task route not found")
}

func (server *Server) handleTradingTaskCollection(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tasks, err := server.repository.ListTradingTasks(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, tasks)
	case http.MethodPost:
		var request data.CreateTradingTask
		if err := readJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		normalizeCreateTradingTask(&request)
		definition, err := server.strategyRepository.GetStrategy(r.Context(), request.StrategyID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		if err := validateCreateTradingTask(request, definition); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task, err := server.repository.CreateTradingTask(r.Context(), request)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, task)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (server *Server) getTradingTask(w http.ResponseWriter, r *http.Request, id string) {
	task, err := server.repository.GetTradingTask(r.Context(), id)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) handleTradingTaskCommand(w http.ResponseWriter, r *http.Request, id string, action string) {
	status := data.TaskStatus("")
	switch action {
	case "start":
		status = data.TaskStatusRunning
	case "pause", "stop":
		status = data.TaskStatusPaused
	default:
		writeError(w, http.StatusNotFound, "trading task command not found")
		return
	}
	task, err := server.repository.SetTradingTaskStatus(r.Context(), id, status)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, task)
}

func (server *Server) handleTradingTaskDetailCollection(
	w http.ResponseWriter,
	r *http.Request,
	id string,
	collection string,
) {
	if _, err := server.repository.GetTradingTask(r.Context(), id); err != nil {
		writeStoreError(w, err)
		return
	}
	switch collection {
	case "intents":
		intents, err := server.repository.ListTradingIntents(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, intents)
	case "orders":
		orders, err := server.repository.ListTradingOrders(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, orders)
	case "notifications":
		notifications, err := server.repository.ListTradingNotifications(r.Context(), id)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, notifications)
	default:
		writeError(w, http.StatusNotFound, "trading task collection not found")
	}
}

func (server *Server) handleStrategies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	parts := pathParts(r.URL.Path)
	switch len(parts) {
	case 2:
		strategies, err := server.strategyRepository.ListStrategies(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, strategies)
	case 3:
		definition, err := server.strategyRepository.GetStrategy(r.Context(), parts[2])
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, definition)
	default:
		writeError(w, http.StatusNotFound, "strategy route not found")
	}
}

func (server *Server) serveFrontend(w http.ResponseWriter, r *http.Request) {
	if server.staticRoot == "" || r.Method != http.MethodGet {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	requestPath := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
	if requestPath == "." {
		requestPath = "index.html"
	}
	if strings.HasPrefix(requestPath, "..") {
		writeError(w, http.StatusBadRequest, "invalid path")
		return
	}
	fullPath := filepath.Join(server.staticRoot, requestPath)
	if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
		http.ServeFile(w, r, fullPath)
		return
	}
	http.ServeFile(w, r, filepath.Join(server.staticRoot, "index.html"))
}

type taskCommand struct {
	id       string
	category string
	action   string
}

func parseCandleQuery(r *http.Request) (data.CandleQuery, error) {
	values := r.URL.Query()
	query := data.CandleQuery{
		Exchange: values.Get("exchange"),
		Symbol:   values.Get("symbol"),
		Interval: values.Get("interval"),
		Limit:    1000,
	}
	if query.Exchange == "" || query.Symbol == "" || query.Interval == "" {
		return data.CandleQuery{}, errors.New("exchange, symbol and interval are required")
	}
	if rawLimit := values.Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return data.CandleQuery{}, errors.New("limit must be a positive integer")
		}
		query.Limit = limit
	}
	from, err := parseOptionalTime(values.Get("from"))
	if err != nil {
		return data.CandleQuery{}, fmt.Errorf("from: %w", err)
	}
	to, err := parseOptionalTime(values.Get("to"))
	if err != nil {
		return data.CandleQuery{}, fmt.Errorf("to: %w", err)
	}
	query.From = from
	query.To = to
	return query, nil
}

func validateCreateTask(task data.CreateDataSyncTask) error {
	if task.Exchange == "" || task.Symbol == "" || task.Interval == "" {
		return errors.New("exchange, symbol and interval are required")
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
	if task.Name == "" || task.Type == "" || task.Exchange == "" || task.Symbol == "" || task.StrategyID == "" {
		return errors.New("name, type, exchange, symbol and strategyId are required")
	}
	if task.Type != "paper" && task.Type != "live" {
		return errors.New("type must be paper or live")
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

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func pathParts(requestPath string) []string {
	trimmed := strings.Trim(requestPath, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func readJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, data.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeError(w, http.StatusInternalServerError, err.Error())
}

func parseOptionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
