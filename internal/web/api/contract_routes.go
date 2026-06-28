package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/lofreer/tictick-hi/internal/data"
)

type operationConfig struct {
	auth        bool
	csrf        bool
	parameters  []apiParameter
	requestBody *apiRequestBody
	errors      []int
}

type operationOption func(*operationConfig)

func apiContractPaths() map[string]apiPathItem {
	paths := map[string]apiPathItem{}
	addHealthContractPaths(paths)
	addAuthContractPaths(paths)
	addDataContractPaths(paths)
	addMarketContractPaths(paths)
	addStrategyContractPaths(paths)
	addBacktestContractPaths(paths)
	addTradingContractPaths(paths)
	addSystemContractPaths(paths)
	return paths
}

func addHealthContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/readyz", http.MethodGet, operation(
		"system", "readyz", "Read API readiness", http.StatusOK, schemaRef("StatusResponse"),
		withPublic(),
	))
}

func addAuthContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/auth/login", http.MethodPost, operation(
		"auth", "login", "Create an operator session", http.StatusOK, schemaRef("Operator"),
		withPublic(), withRequest(schemaRef("LoginRequest")), withErrors(http.StatusBadRequest, http.StatusUnauthorized, http.StatusTooManyRequests),
	))
	addOperation(paths, "/api/auth/me", http.MethodGet, operation(
		"auth", "currentOperator", "Read the current operator", http.StatusOK, schemaRef("Operator"),
	))
	addOperation(paths, "/api/auth/logout", http.MethodPost, operation(
		"auth", "logout", "Revoke the current operator session", http.StatusOK, schemaRef("StatusResponse"),
		withCSRF(),
	))
	addOperation(paths, "/api/auth/sessions", http.MethodGet, operation(
		"auth", "listOperatorSessions", "List current operator sessions", http.StatusOK, arraySchema(schemaRef("OperatorSession")),
	))
	addOperation(paths, "/api/auth/sessions/{id}", http.MethodDelete, operation(
		"auth", "revokeOperatorSession", "Revoke another operator session", http.StatusOK, schemaRef("StatusResponse"),
		withCSRF(), withParameters(pathParam("id", "Operator session id")), withErrors(http.StatusNotFound),
	))
}

func addDataContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/data/tasks", http.MethodGet, operation(
		"data", "listDataSyncTasks", "List data sync tasks", http.StatusOK, arraySchema(schemaRef("DataSyncTask")),
	))
	addOperation(paths, "/api/data/tasks", http.MethodPost, operation(
		"data", "createDataSyncTask", "Create a data sync task", http.StatusCreated, schemaRef("DataSyncTask"),
		withCSRF(), withRequest(schemaRef("CreateDataSyncTask")), withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/data/tasks/{id}", http.MethodDelete, operation(
		"data", "deleteDataSyncTask", "Delete a data sync task", http.StatusNoContent, nil,
		withCSRF(), withParameters(pathParam("id", "Data sync task id")), withErrors(http.StatusNotFound),
	))
	addOperation(paths, "/api/data/tasks/{id}/retry", http.MethodPost, operation(
		"data", "retryDataSyncTask", "Retry a failed data sync task", http.StatusOK, schemaRef("DataSyncTask"),
		withCSRF(), withParameters(pathParam("id", "Data sync task id")), withErrors(http.StatusNotFound, http.StatusConflict),
	))
	addOperation(paths, "/api/data/tasks/{id}/gaps", http.MethodGet, operation(
		"data", "listDataSyncTaskGaps", "List detected data sync task gaps", http.StatusOK, schemaRef("DataSyncGapList"),
		withParameters(pathParam("id", "Data sync task id")), withErrors(http.StatusNotFound),
	))
	addOperation(paths, "/api/data/tasks/{id}/repair-gaps", http.MethodPost, operation(
		"data", "repairDataSyncTaskGaps", "Create sync tasks for detected task gaps", http.StatusOK, schemaRef("DataSyncGapRepairResult"),
		withCSRF(), withParameters(pathParam("id", "Data sync task id")), withErrors(http.StatusNotFound),
	))
	addOperation(paths, "/api/data/tasks/{id}/repair-gap", http.MethodPost, operation(
		"data", "repairDataSyncTaskGap", "Create a sync task for one chart gap", http.StatusOK, schemaRef("DataSyncGapRepairResult"),
		withCSRF(), withParameters(pathParam("id", "Data sync task id")), withRequest(schemaRef("RepairDataSyncTaskGapRequest")), withErrors(http.StatusBadRequest, http.StatusNotFound),
	))
	addOperation(paths, "/api/data/tasks/{id}/sync/{action}", http.MethodPost, operation(
		"data", "setDataSyncEnabled", "Start or stop historical data sync", http.StatusOK, schemaRef("DataSyncTask"),
		withCSRF(), withParameters(pathParam("id", "Data sync task id"), enumPathParam("action", "Sync action", "start", "stop")),
		withErrors(http.StatusNotFound, http.StatusConflict),
	))
	addOperation(paths, "/api/data/tasks/{id}/realtime/{action}", http.MethodPost, operation(
		"data", "setDataRealtimeEnabled", "Start or stop realtime data sync", http.StatusOK, schemaRef("DataSyncTask"),
		withCSRF(), withParameters(pathParam("id", "Data sync task id"), enumPathParam("action", "Realtime action", "start", "stop")),
		withErrors(http.StatusNotFound, http.StatusConflict),
	))
	addOperation(paths, "/api/candles", http.MethodGet, operation(
		"data", "getCandles", "Read candles with source and health metadata", http.StatusOK, schemaRef("CandleResult"),
		withParameters(candleQueryParameters()...), withErrors(http.StatusBadRequest),
	))
}

func addMarketContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/market/candle-gaps", http.MethodGet, operation(
		"market", "scanMarketCandleGaps", "Scan persisted market candle gaps", http.StatusOK, schemaRef("MarketCandleGapScan"),
		withParameters(
			queryParam("exchange", true, "Exchange id", map[string]any{"type": "string", "enum": []string{"binance", "okx"}}),
			queryParam("symbol", true, "Market symbol", map[string]any{"type": "string"}),
			queryParam("interval", true, "Candle interval", map[string]any{"type": "string", "enum": []string{"1m", "5m", "15m", "1h", "4h", "1d"}}),
			queryParam("limit", false, "Maximum returned gaps", map[string]any{"type": "integer", "minimum": 1, "maximum": data.MaxMarketCandleGapScanLimit}),
		),
		withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/market/candle-gaps/repair", http.MethodPost, operation(
		"market", "repairMarketCandleGap", "Create a sync task for one persisted market gap", http.StatusOK, schemaRef("DataSyncGapRepairResult"),
		withCSRF(), withRequest(schemaRef("RepairMarketCandleGapRequest")), withErrors(http.StatusBadRequest, http.StatusNotFound),
	))
	addOperation(paths, "/api/market/candle-gaps/repair-batch", http.MethodPost, operation(
		"market", "repairMarketCandleGaps", "Create sync tasks for persisted market gaps", http.StatusOK, schemaRef("DataSyncGapRepairResult"),
		withCSRF(), withRequest(schemaRef("RepairMarketCandleGapsRequest")), withErrors(http.StatusBadRequest, http.StatusNotFound),
	))
	addOperation(paths, "/api/market/instruments", http.MethodGet, operation(
		"market", "listMarketInstruments", "Search market instruments", http.StatusOK, arraySchema(schemaRef("MarketInstrument")),
		withParameters(
			queryParam("exchange", true, "Exchange id", map[string]any{"type": "string", "enum": []string{"binance", "okx"}}),
			queryParam("q", false, "Search text", map[string]any{"type": "string"}),
			queryParam("limit", false, "Maximum number of instruments", map[string]any{"type": "integer", "minimum": 1, "maximum": 50}),
		),
		withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/market/instruments/sync", http.MethodPost, operation(
		"market", "syncMarketInstruments", "Synchronize market instruments", http.StatusOK, marketInstrumentSyncResultSchema(),
		withCSRF(), withParameters(
			queryParam("exchange", true, "Exchange id", map[string]any{"type": "string", "enum": []string{"binance", "okx"}}),
		),
		withErrors(http.StatusBadRequest),
	))
}

func marketInstrumentSyncResultSchema() map[string]any {
	return map[string]any{
		"type":     "object",
		"required": []string{"exchange", "activeCount", "inactiveCount", "syncedAt"},
		"properties": map[string]any{
			"exchange":      map[string]any{"type": "string"},
			"activeCount":   map[string]any{"type": "integer"},
			"inactiveCount": map[string]any{"type": "integer"},
			"syncedAt":      map[string]any{"type": "string", "format": "date-time"},
		},
	}
}

func addStrategyContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/strategies", http.MethodGet, operation(
		"strategies", "listStrategies", "List strategy definitions", http.StatusOK, arraySchema(schemaRef("StrategyDefinition")),
	))
	addOperation(paths, "/api/strategies/{id}", http.MethodGet, operation(
		"strategies", "getStrategy", "Read one strategy definition", http.StatusOK, schemaRef("StrategyDefinition"),
		withParameters(pathParam("id", "Strategy id")), withErrors(http.StatusNotFound),
	))
}

func addBacktestContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/backtests", http.MethodGet, operation(
		"backtests", "listBacktests", "List backtest tasks", http.StatusOK, arraySchema(schemaRef("BacktestTask")),
	))
	addOperation(paths, "/api/backtests", http.MethodPost, operation(
		"backtests", "createBacktest", "Create a backtest task", http.StatusCreated, schemaRef("BacktestTask"),
		withCSRF(), withRequest(schemaRef("CreateBacktestTask")), withErrors(http.StatusBadRequest, http.StatusNotFound),
	))
	addOperation(paths, "/api/backtests/{id}", http.MethodGet, operation(
		"backtests", "getBacktest", "Read a backtest task", http.StatusOK, schemaRef("BacktestTask"),
		withParameters(pathParam("id", "Backtest task id")), withErrors(http.StatusNotFound),
	))
	addOperation(paths, "/api/backtests/{id}/orders", http.MethodGet, operation(
		"backtests", "listBacktestOrders", "List backtest orders", http.StatusOK, arraySchema(schemaRef("BacktestOrder")),
		withParameters(pathParam("id", "Backtest task id")), withErrors(http.StatusNotFound),
	))
	addOperation(paths, "/api/backtests/{id}/intents", http.MethodGet, operation(
		"backtests", "listBacktestIntents", "List backtest strategy intents", http.StatusOK, arraySchema(schemaRef("StrategyIntent")),
		withParameters(pathParam("id", "Backtest task id")), withErrors(http.StatusNotFound),
	))
}

func addTradingContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/trading/tasks", http.MethodGet, operation(
		"trading", "listTradingTasks", "List trading tasks", http.StatusOK, arraySchema(schemaRef("TradingTask")),
	))
	addOperation(paths, "/api/trading/tasks", http.MethodPost, operation(
		"trading", "createTradingTask", "Create a paper or live trading task", http.StatusCreated, schemaRef("TradingTask"),
		withCSRF(), withRequest(schemaRef("CreateTradingTask")), withErrors(http.StatusBadRequest, http.StatusNotFound),
	))
	addOperation(paths, "/api/trading/tasks/{id}", http.MethodGet, operation(
		"trading", "getTradingTask", "Read a trading task", http.StatusOK, schemaRef("TradingTask"),
		withParameters(pathParam("id", "Trading task id")), withErrors(http.StatusNotFound),
	))
	for _, action := range []string{"start", "pause", "stop"} {
		addOperation(paths, "/api/trading/tasks/{id}/"+action, http.MethodPost, operation(
			"trading", "setTradingTask"+titleWord(action), "Set trading task status to "+action, http.StatusOK, schemaRef("TradingTask"),
			withCSRF(), withParameters(pathParam("id", "Trading task id")), withErrors(http.StatusNotFound, http.StatusConflict),
		))
	}
	for _, collection := range []struct {
		name    string
		model   string
		opID    string
		summary string
	}{
		{"intents", "StrategyIntent", "listTradingIntents", "List trading strategy intents"},
		{"orders", "Order", "listTradingOrders", "List trading orders"},
		{"executions", "Execution", "listTradingExecutions", "List trading executions"},
		{"positions", "Position", "listTradingPositions", "List trading positions"},
		{"notifications", "Notification", "listTradingNotifications", "List trading notifications"},
	} {
		addOperation(paths, "/api/trading/tasks/{id}/"+collection.name, http.MethodGet, operation(
			"trading", collection.opID, collection.summary, http.StatusOK, arraySchema(schemaRef(collection.model)),
			withParameters(pathParam("id", "Trading task id")), withErrors(http.StatusNotFound),
		))
	}
}

func addSystemContractPaths(paths map[string]apiPathItem) {
	addOperation(paths, "/api/system/health", http.MethodGet, operation(
		"system", "systemHealth", "Read system health", http.StatusOK, schemaRef("SystemHealth"),
	))
	addOperation(paths, "/api/system/api-contract", http.MethodGet, operation(
		"system", "apiContract", "Read the OpenAPI schema contract", http.StatusOK,
		map[string]any{"type": "object", "description": "OpenAPI 3.1 contract document"},
	))
	addOperation(paths, "/api/system/notifications", http.MethodGet, operation(
		"system", "listNotifications", "List notifications", http.StatusOK, arraySchema(schemaRef("Notification")),
	))
	addOperation(paths, "/api/system/notifications/{id}/retry", http.MethodPost, operation(
		"system", "retryNotification", "Retry a failed notification", http.StatusOK, schemaRef("Notification"),
		withCSRF(), withParameters(pathParam("id", "Notification id")), withErrors(http.StatusNotFound, http.StatusConflict),
	))
	addOperation(paths, "/api/system/notifications/channels", http.MethodGet, operation(
		"system", "listNotificationChannels", "List notification channels", http.StatusOK, arraySchema(schemaRef("NotificationChannel")),
	))
	addOperation(paths, "/api/system/notifications/channels", http.MethodPost, operation(
		"system", "createNotificationChannel", "Create a notification channel", http.StatusCreated, schemaRef("NotificationChannel"),
		withCSRF(), withRequest(schemaRef("CreateNotificationChannel")), withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/system/exchange-accounts", http.MethodGet, operation(
		"system", "listExchangeAccounts", "List exchange accounts without secret material", http.StatusOK, arraySchema(schemaRef("ExchangeAccount")),
	))
	addOperation(paths, "/api/system/exchange-accounts", http.MethodPost, operation(
		"system", "createExchangeAccount", "Create an exchange account", http.StatusCreated, schemaRef("ExchangeAccount"),
		withCSRF(), withRequest(schemaRef("CreateExchangeAccount")), withErrors(http.StatusBadRequest),
	))
	addOperation(paths, "/api/system/operators", http.MethodGet, operation(
		"system", "listOperators", "List operators", http.StatusOK, arraySchema(schemaRef("Operator")),
	))
	addOperation(paths, "/api/system/operators", http.MethodPost, operation(
		"system", "createOperator", "Create an operator", http.StatusCreated, schemaRef("Operator"),
		withCSRF(), withRequest(schemaRef("CreateOperator")), withErrors(http.StatusBadRequest),
	))
	for _, action := range []string{"enable", "disable"} {
		addOperation(paths, "/api/system/operators/{id}/"+action, http.MethodPost, operation(
			"system", "setOperator"+titleWord(action), "Set operator "+action, http.StatusOK, schemaRef("Operator"),
			withCSRF(), withParameters(pathParam("id", "Operator id")), withErrors(http.StatusNotFound),
		))
	}
	addOperation(paths, "/api/system/audit-events", http.MethodGet, operation(
		"system", "listAuditEvents", "List operation audit events", http.StatusOK, arraySchema(schemaRef("AuditEvent")),
		withParameters(queryParam("limit", false, "Maximum number of events", map[string]any{"type": "integer", "minimum": 1, "maximum": 500})),
	))
}

func addOperation(paths map[string]apiPathItem, path string, method string, operation apiOperation) {
	if paths[path] == nil {
		paths[path] = apiPathItem{}
	}
	paths[path][strings.ToLower(method)] = operation
}

func operation(tag string, id string, summary string, status int, responseSchema map[string]any, options ...operationOption) apiOperation {
	config := operationConfig{auth: true}
	for _, option := range options {
		option(&config)
	}
	return apiOperation{
		Tags:        []string{tag},
		Summary:     summary,
		OperationID: id,
		Parameters:  config.parameters,
		RequestBody: config.requestBody,
		Responses:   operationResponses(status, responseSchema, config),
		Security:    operationSecurity(config),
	}
}

func operationResponses(status int, responseSchema map[string]any, config operationConfig) map[string]apiResponse {
	responses := map[string]apiResponse{
		strconv.Itoa(status): successResponse(status, responseSchema),
	}
	for _, code := range errorCodes(config) {
		responses[strconv.Itoa(code)] = errorResponse(code)
	}
	return responses
}

func successResponse(status int, schema map[string]any) apiResponse {
	if status == http.StatusNoContent {
		return apiResponse{Description: "No content"}
	}
	return jsonResponse(http.StatusText(status), schema)
}

func jsonResponse(description string, schema map[string]any) apiResponse {
	return apiResponse{
		Description: description,
		Content:     map[string]apiMediaType{jsonMediaType: {Schema: schema}},
	}
}

func errorResponse(status int) apiResponse {
	response := jsonResponse(errorDescription(status), schemaRef("APIErrorResponse"))
	response.XErrorCodes = apiErrorCodesForHTTPStatus(status)
	return response
}

func errorCodes(config operationConfig) []int {
	codes := []int{http.StatusMethodNotAllowed, http.StatusInternalServerError}
	if config.auth {
		codes = append(codes, http.StatusUnauthorized)
	}
	if config.csrf {
		codes = append(codes, http.StatusForbidden)
	}
	return uniqueStatusCodes(append(codes, config.errors...))
}

func uniqueStatusCodes(codes []int) []int {
	seen := map[int]bool{}
	result := make([]int, 0, len(codes))
	for _, code := range codes {
		if !seen[code] {
			result = append(result, code)
			seen[code] = true
		}
	}
	return result
}

func errorDescription(code int) string {
	if text := http.StatusText(code); text != "" {
		return text
	}
	return "Error"
}

func operationSecurity(config operationConfig) []map[string][]string {
	if !config.auth {
		return nil
	}
	requirement := map[string][]string{"sessionCookie": []string{}}
	if config.csrf {
		requirement["csrfHeader"] = []string{}
	}
	return []map[string][]string{requirement}
}

func titleWord(value string) string {
	if value == "" {
		return value
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func withPublic() operationOption {
	return func(config *operationConfig) {
		config.auth = false
		config.csrf = false
	}
}

func withCSRF() operationOption {
	return func(config *operationConfig) {
		config.csrf = true
	}
}

func withRequest(schema map[string]any) operationOption {
	return func(config *operationConfig) {
		config.requestBody = &apiRequestBody{
			Required: true,
			Content:  map[string]apiMediaType{jsonMediaType: {Schema: schema}},
		}
	}
}

func withParameters(parameters ...apiParameter) operationOption {
	return func(config *operationConfig) {
		config.parameters = append(config.parameters, parameters...)
	}
}

func withErrors(codes ...int) operationOption {
	return func(config *operationConfig) {
		config.errors = append(config.errors, codes...)
	}
}

func pathParam(name string, description string) apiParameter {
	return apiParameter{Name: name, In: "path", Required: true, Description: description, Schema: map[string]any{"type": "string"}}
}

func enumPathParam(name string, description string, values ...string) apiParameter {
	return apiParameter{Name: name, In: "path", Required: true, Description: description, Schema: map[string]any{"type": "string", "enum": values}}
}

func queryParam(name string, required bool, description string, schema map[string]any) apiParameter {
	return apiParameter{Name: name, In: "query", Required: required, Description: description, Schema: schema}
}

func candleQueryParameters() []apiParameter {
	intervals := []string{"1m", "5m", "15m", "1h", "4h", "1d"}
	return []apiParameter{
		queryParam("exchange", true, "Exchange id", map[string]any{"type": "string"}),
		queryParam("symbol", true, "Trading symbol", map[string]any{"type": "string"}),
		queryParam("interval", true, "Requested candle interval", map[string]any{"type": "string", "enum": intervals}),
		queryParam("from", false, "Inclusive start time", map[string]any{"type": "string", "format": "date-time"}),
		queryParam("to", false, "Inclusive end time", map[string]any{"type": "string", "format": "date-time"}),
		queryParam("limit", false, "Maximum candle count", map[string]any{"type": "integer", "minimum": 1, "maximum": data.MaxCandleLimit}),
		queryParam("cursor", false, "Opaque adjacent-window cursor returned by CandlePagination", map[string]any{"type": "string"}),
	}
}
