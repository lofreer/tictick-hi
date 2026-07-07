package api

import (
	"reflect"
	"strings"
	"time"

	"github.com/lofreer/tictick-hi/internal/data"
	"github.com/lofreer/tictick-hi/internal/strategy"
)

const jsonMediaType = "application/json"

type openAPIContract struct {
	OpenAPI    string                 `json:"openapi"`
	Info       apiContractInfo        `json:"info"`
	Paths      map[string]apiPathItem `json:"paths"`
	Components apiContractComponents  `json:"components"`
}

type apiContractInfo struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type apiContractComponents struct {
	Schemas         map[string]map[string]any    `json:"schemas"`
	SecuritySchemes map[string]apiSecurityScheme `json:"securitySchemes"`
	ErrorCodes      []apiErrorDefinition         `json:"x-errorCodes"`
}

type apiSecurityScheme struct {
	Type string `json:"type"`
	In   string `json:"in"`
	Name string `json:"name"`
}

type apiPathItem map[string]apiOperation

type apiOperation struct {
	Tags        []string               `json:"tags,omitempty"`
	Summary     string                 `json:"summary"`
	OperationID string                 `json:"operationId"`
	Parameters  []apiParameter         `json:"parameters,omitempty"`
	RequestBody *apiRequestBody        `json:"requestBody,omitempty"`
	Responses   map[string]apiResponse `json:"responses"`
	Security    []map[string][]string  `json:"security,omitempty"`
}

type apiParameter struct {
	Name        string         `json:"name"`
	In          string         `json:"in"`
	Required    bool           `json:"required"`
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
}

type apiRequestBody struct {
	Required bool                    `json:"required"`
	Content  map[string]apiMediaType `json:"content"`
}

type apiResponse struct {
	Description string                  `json:"description"`
	Headers     map[string]apiHeader    `json:"headers,omitempty"`
	Content     map[string]apiMediaType `json:"content,omitempty"`
	XErrorCodes []string                `json:"x-errorCodes,omitempty"`
}

type apiHeader struct {
	Description string         `json:"description,omitempty"`
	Schema      map[string]any `json:"schema"`
}

func requestIDResponseHeader() map[string]apiHeader {
	return map[string]apiHeader{
		requestIDHeaderName: {
			Description: "HTTP request correlation id",
			Schema:      map[string]any{"type": "string"},
		},
		traceparentHeaderName: {
			Description: "W3C trace context",
			Schema:      map[string]any{"type": "string"},
		},
	}
}

type apiMediaType struct {
	Schema map[string]any `json:"schema"`
}

type apiStatusResponse struct {
	Status string `json:"status"`
}

type contractModel struct {
	name  string
	value any
}

func apiContractDocument() openAPIContract {
	return openAPIContract{
		OpenAPI: "3.1.0",
		Info: apiContractInfo{
			Title:   "tictick-hi API",
			Version: "stage8-schema-contract",
		},
		Paths: apiContractPaths(),
		Components: apiContractComponents{
			Schemas:         buildContractSchemas(),
			SecuritySchemes: apiContractSecuritySchemes(),
			ErrorCodes:      apiErrorCatalog(),
		},
	}
}

func buildContractSchemas() map[string]map[string]any {
	registry := newSchemaRegistry([]contractModel{
		{"APIErrorResponse", apiErrorResponse{}},
		{"StatusResponse", apiStatusResponse{}},
		{"TaskStatus", data.TaskStatus("")},
		{"DataSyncHealth", data.DataSyncHealth("")},
		{"DataSyncMarketStatus", data.DataSyncMarketStatus("")},
		{"DataSyncGapSummary", data.DataSyncGapSummary{}},
		{"DataSyncInvalidSummary", data.DataSyncInvalidSummary{}},
		{"DataSyncGapList", data.DataSyncGapList{}},
		{"DataSyncInvalidIssueList", data.DataSyncInvalidIssueList{}},
		{"DataSyncGapRepairResult", data.DataSyncGapRepairResult{}},
		{"RepairDataSyncTaskGapRequest", data.RepairDataSyncTaskGapRequest{}},
		{"RepairDataSyncInvalidIssuesRequest", data.RepairDataSyncInvalidIssuesRequest{}},
		{"RepairMarketCandleGapRequest", data.RepairMarketCandleGapRequest{}},
		{"RepairMarketCandleGapWindow", data.RepairMarketCandleGapWindow{}},
		{"RepairMarketCandleGapsRequest", data.RepairMarketCandleGapsRequest{}},
		{"RepairMarketCandleInvalidIssuesRequest", data.RepairMarketCandleInvalidIssuesRequest{}},
		{"QuarantineMarketCandleInvalidIssuesRequest", data.QuarantineMarketCandleInvalidIssuesRequest{}},
		{"MarketCandleQuarantineRecord", data.MarketCandleQuarantineRecord{}},
		{"MarketCandleQuarantineResult", data.MarketCandleQuarantineResult{}},
		{"DataSyncTask", data.DataSyncTask{}},
		{"CreateDataSyncTask", data.CreateDataSyncTask{}},
		{"Candle", data.Candle{}},
		{"CandleSource", data.CandleSource("")},
		{"CandleHealth", data.CandleHealth("")},
		{"CandleGap", data.CandleGap{}},
		{"CandleIssue", data.CandleIssue{}},
		{"CandleCoverage", data.CandleCoverage{}},
		{"CandleWindow", data.CandleWindow{}},
		{"CandlePagination", data.CandlePagination{}},
		{"CandleResult", data.CandleResult{}},
		{"MarketCandleGapScan", data.MarketCandleGapScan{}},
		{"MarketCandleInvalidIssueScan", data.MarketCandleInvalidIssueScan{}},
		{"MarketInstrument", data.MarketInstrument{}},
		{"MarketInstrumentSyncStatus", data.MarketInstrumentSyncStatus{}},
		{"OverviewRecentFacts", data.OverviewRecentFacts{}},
		{"OverviewStrategyIntentFact", data.OverviewStrategyIntentFact{}},
		{"OverviewOrderFact", data.OverviewOrderFact{}},
		{"OverviewTrends", data.OverviewTrends{}},
		{"OverviewTrendBucket", data.OverviewTrendBucket{}},
		{"StrategyDefinition", strategy.Definition{}},
		{"StrategyParamSpec", strategy.ParamSpec{}},
		{"StrategyOption", strategy.Option{}},
		{"BacktestTask", data.BacktestTask{}},
		{"CreateBacktestTask", data.CreateBacktestTask{}},
		{"BacktestOrder", data.BacktestOrder{}},
		{"StrategyIntent", data.StrategyIntent{}},
		{"TradingTask", data.TradingTask{}},
		{"CreateTradingTask", data.CreateTradingTask{}},
		{"Order", data.Order{}},
		{"Execution", data.Execution{}},
		{"Position", data.Position{}},
		{"Notification", data.Notification{}},
		{"NotificationChannel", data.NotificationChannel{}},
		{"CreateNotificationChannel", data.CreateNotificationChannel{}},
		{"ExchangeAccount", data.ExchangeAccount{}},
		{"CreateExchangeAccount", data.CreateExchangeAccount{}},
		{"Operator", data.Operator{}},
		{"CreateOperator", data.CreateOperator{}},
		{"LoginRequest", data.LoginRequest{}},
		{"OperatorSession", data.OperatorSession{}},
		{"SystemHealth", data.SystemHealth{}},
		{"ServiceHealth", data.ServiceHealth{}},
		{"AuditEvent", data.AuditEvent{}},
		{"AuditEventPage", data.AuditEventPage{}},
		{"AuditEventHashChainVerification", data.AuditEventHashChainVerification{}},
	})
	schemas := registry.schemas()
	schemas["APIErrorCode"] = apiErrorCodeSchema()
	if properties, ok := schemas["APIErrorResponse"]["properties"].(map[string]any); ok {
		properties["code"] = schemaRef("APIErrorCode")
	}
	return schemas
}

func apiContractSecuritySchemes() map[string]apiSecurityScheme {
	return map[string]apiSecurityScheme{
		"sessionCookie": {Type: "apiKey", In: "cookie", Name: sessionCookieName},
		"csrfHeader":    {Type: "apiKey", In: "header", Name: csrfHeaderName},
	}
}

type schemaRegistry struct {
	models []contractModel
	names  map[reflect.Type]string
}

func newSchemaRegistry(models []contractModel) schemaRegistry {
	names := make(map[reflect.Type]string, len(models))
	for _, model := range models {
		names[indirectType(reflect.TypeOf(model.value))] = model.name
	}
	return schemaRegistry{models: models, names: names}
}

func (registry schemaRegistry) schemas() map[string]map[string]any {
	schemas := make(map[string]map[string]any, len(registry.models))
	for _, model := range registry.models {
		schemas[model.name] = registry.schemaForComponent(indirectType(reflect.TypeOf(model.value)))
	}
	return schemas
}

func (registry schemaRegistry) schemaForComponent(t reflect.Type) map[string]any {
	if schema := enumSchema(t); schema != nil {
		return schema
	}
	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	if t.Kind() != reflect.Struct {
		return registry.schemaForType(t)
	}
	return registry.schemaForStruct(t)
}

func (registry schemaRegistry) schemaForType(t reflect.Type) map[string]any {
	t = indirectType(t)
	if name, ok := registry.names[t]; ok {
		return schemaRef(name)
	}
	if schema := enumSchema(t); schema != nil {
		return schema
	}
	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer", "minimum": 0}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		return arraySchema(registry.schemaForType(t.Elem()))
	case reflect.Map:
		return mapSchema(registry.schemaForType(t.Elem()))
	case reflect.Interface:
		return map[string]any{}
	case reflect.Struct:
		return registry.schemaForStruct(t)
	default:
		return map[string]any{}
	}
}

func (registry schemaRegistry) schemaForStruct(t reflect.Type) map[string]any {
	properties := map[string]any{}
	required := make([]string, 0, t.NumField())
	for index := 0; index < t.NumField(); index++ {
		field := t.Field(index)
		name, omitEmpty, ok := jsonField(field)
		if !ok {
			continue
		}
		properties[name] = registry.schemaForType(field.Type)
		if !omitEmpty {
			required = append(required, name)
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func jsonField(field reflect.StructField) (string, bool, bool) {
	if field.PkgPath != "" {
		return "", false, false
	}
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" {
		name = lowerFirst(field.Name)
	}
	return name, contains(parts[1:], "omitempty"), true
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func indirectType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func enumSchema(t reflect.Type) map[string]any {
	switch t {
	case reflect.TypeOf(data.TaskStatus("")):
		return map[string]any{"type": "string", "enum": []string{
			string(data.TaskStatusPending),
			string(data.TaskStatusRunning),
			string(data.TaskStatusStopping),
			string(data.TaskStatusPaused),
			string(data.TaskStatusSucceeded),
			string(data.TaskStatusFailed),
			string(data.TaskStatusCancelled),
		}}
	case reflect.TypeOf(data.DataSyncHealth("")):
		return map[string]any{"type": "string", "enum": []string{
			string(data.DataSyncHealthOK),
			string(data.DataSyncHealthSyncing),
			string(data.DataSyncHealthGap),
			string(data.DataSyncHealthFailed),
			string(data.DataSyncHealthPaused),
			string(data.DataSyncHealthRetrying),
			string(data.DataSyncHealthInsufficient),
			string(data.DataSyncHealthInvalid),
		}}
	case reflect.TypeOf(data.DataSyncMarketStatus("")):
		return map[string]any{"type": "string", "enum": []string{
			string(data.DataSyncMarketStatusActive),
			string(data.DataSyncMarketStatusInactive),
			string(data.DataSyncMarketStatusMissing),
		}}
	case reflect.TypeOf(data.CandleSource("")):
		return map[string]any{"type": "string", "enum": []string{
			string(data.CandleSourceNative),
			string(data.CandleSourceAggregated),
			string(data.CandleSourceNone),
		}}
	case reflect.TypeOf(data.CandleHealth("")):
		return map[string]any{"type": "string", "enum": []string{
			string(data.CandleHealthOK),
			string(data.CandleHealthGap),
			string(data.CandleHealthInsufficient),
			string(data.CandleHealthInvalid),
		}}
	default:
		return nil
	}
}

func schemaRef(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func arraySchema(item map[string]any) map[string]any {
	return map[string]any{"type": "array", "items": item}
}

func mapSchema(value map[string]any) map[string]any {
	return map[string]any{"type": "object", "additionalProperties": value}
}
