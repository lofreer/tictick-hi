package api

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

type tsObjectType struct {
	Name   string
	Fields map[string]tsField
}

type tsField struct {
	Name     string
	Optional bool
}

func TestFrontendAPIRequestTypesMatchContractSchemas(t *testing.T) {
	types := parseTSTypesFile(t, generatedAPITypesPath(t))
	schemas := apiContractDocument().Components.Schemas

	for _, item := range []struct {
		tsType     string
		schemaName string
	}{
		{"LoginRequest", "LoginRequest"},
		{"CreateDataSyncTask", "CreateDataSyncTask"},
		{"RepairMarketCandleGapRequest", "RepairMarketCandleGapRequest"},
		{"CreateBacktestTask", "CreateBacktestTask"},
		{"CreateTradingTask", "CreateTradingTask"},
		{"CreateNotificationChannel", "CreateNotificationChannel"},
		{"CreateExchangeAccount", "CreateExchangeAccount"},
		{"CreateOperator", "CreateOperator"},
	} {
		assertTSFieldsEqualSchema(t, types[item.tsType], item.schemaName, schemas[item.schemaName])
		assertTSOptionalityMatchesSchema(t, types[item.tsType], item.schemaName, schemas[item.schemaName])
	}
}

func TestFrontendAPIResponseTypesMatchContractFields(t *testing.T) {
	types := parseTSTypesFile(t, generatedAPITypesPath(t))
	schemas := apiContractDocument().Components.Schemas

	for _, item := range []struct {
		tsType     string
		schemaName string
	}{
		{"DataSyncTask", "DataSyncTask"},
		{"DataSyncGapSummary", "DataSyncGapSummary"},
		{"DataSyncGapList", "DataSyncGapList"},
		{"DataSyncGapRepairResult", "DataSyncGapRepairResult"},
		{"MarketCandleGapScan", "MarketCandleGapScan"},
		{"CandleGap", "CandleGap"},
		{"CandleCoverage", "CandleCoverage"},
		{"CandleResult", "CandleResult"},
		{"BacktestTask", "BacktestTask"},
		{"BacktestOrder", "BacktestOrder"},
		{"StrategyIntent", "StrategyIntent"},
		{"TradingTask", "TradingTask"},
		{"Order", "Order"},
		{"Execution", "Execution"},
		{"Position", "Position"},
		{"Notification", "Notification"},
		{"NotificationChannel", "NotificationChannel"},
		{"ExchangeAccount", "ExchangeAccount"},
		{"Operator", "Operator"},
		{"OperatorSession", "OperatorSession"},
		{"AuditEvent", "AuditEvent"},
		{"ServiceHealth", "ServiceHealth"},
		{"SystemHealth", "SystemHealth"},
		{"StrategyOption", "StrategyOption"},
		{"StrategyParamSpec", "StrategyParamSpec"},
		{"StrategyDefinition", "StrategyDefinition"},
	} {
		assertTSFieldsEqualSchema(t, types[item.tsType], item.schemaName, schemas[item.schemaName])
	}
}

func TestFrontendAPIAdapterResponseFieldsExistInContract(t *testing.T) {
	types := parseTSTypesFile(t, generatedAPITypesPath(t))
	schemas := apiContractDocument().Components.Schemas

	assertTSFieldsEqualSchema(t, types["DataSyncTask"], "DataSyncTask", schemas["DataSyncTask"])
	assertTSFieldsEqualSchema(t, types["CandleResult"], "CandleResult", schemas["CandleResult"])
	assertTSFieldsExistInSchema(t, types["Candle"], "Candle", schemas["Candle"])
}

func TestFrontendAPIAppTypesReferenceGeneratedContract(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(repoRoot(t), "web", "frontend", "src", "types", "app.ts"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`from "@/types/api.generated"`,
		"export type DataSyncTask = APIDataSyncTask;",
		"export type DataSyncGapList = APIDataSyncGapList;",
		"export type MarketCandleGapScan = APIMarketCandleGapScan;",
		"export type RepairMarketCandleGapRequest = APIRepairMarketCandleGapRequest;",
		"export type CreateDataSyncTask = APICreateDataSyncTask;",
		"export type BacktestTask = APIBacktestTask;",
		"export type TradingTask = Omit<APITradingTask",
		"export type Notification = APINotification;",
		"export type SystemHealth = APISystemHealth;",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("app.ts does not reference generated API type %q", expected)
		}
	}
}

func TestFrontendAPICandleQueryMatchesContractParameters(t *testing.T) {
	types := parseTSTypesFile(t, filepath.Join(repoRoot(t), "web", "frontend", "src", "services", "api", "data.ts"))
	queryType := requireTSType(t, types["CandleQuery"], "CandleQuery")
	parameters := queryParametersByName(apiContractDocument().Paths["/api/candles"]["get"].Parameters)

	assertStringSetsEqual(t, "CandleQuery fields", sortedTSFieldNames(queryType), sortedMapKeys(parameters))
	for name, field := range queryType.Fields {
		parameter, ok := parameters[name]
		if !ok {
			t.Fatalf("CandleQuery field %s is missing from /api/candles parameters", name)
		}
		if field.Optional == parameter.Required {
			t.Fatalf("CandleQuery field %s optional=%t, contract required=%t", name, field.Optional, parameter.Required)
		}
	}
}

func parseTSTypesFile(t *testing.T, path string) map[string]tsObjectType {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	types, err := parseTSTypes(string(content))
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return types
}

func parseTSTypes(content string) (map[string]tsObjectType, error) {
	matches := tsObjectTypeExpression.FindAllStringSubmatch(content, -1)
	types := make(map[string]tsObjectType, len(matches))
	for _, match := range matches {
		name := match[1]
		fields := map[string]tsField{}
		for _, field := range strings.Split(match[2], ";") {
			trimmed := strings.TrimSpace(field)
			if trimmed == "" {
				continue
			}
			fieldMatch := tsFieldExpression.FindStringSubmatch(trimmed)
			if len(fieldMatch) == 0 {
				return nil, fmt.Errorf("unsupported field syntax in %s: %q", name, trimmed)
			}
			fieldName := fieldMatch[1]
			fields[fieldName] = tsField{
				Name:     fieldName,
				Optional: fieldMatch[2] == "?",
			}
		}
		types[name] = tsObjectType{Name: name, Fields: fields}
	}
	return types, nil
}

var tsObjectTypeExpression = regexp.MustCompile(`(?s)(?:export\s+)?type\s+([A-Za-z][A-Za-z0-9]*)\s*=\s*\{(.*?)\};`)
var tsFieldExpression = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9]*)(\?)?:\s*.+$`)

func assertTSFieldsEqualSchema(t *testing.T, tsType tsObjectType, schemaName string, schema map[string]any) {
	t.Helper()
	tsType = requireTSType(t, tsType, schemaName)
	assertStringSetsEqual(t, tsType.Name+" fields", sortedTSFieldNames(tsType), sortedMapKeys(schemaProperties(t, schema)))
}

func assertTSFieldsExistInSchema(t *testing.T, tsType tsObjectType, schemaName string, schema map[string]any) {
	t.Helper()
	tsType = requireTSType(t, tsType, schemaName)
	properties := schemaProperties(t, schema)
	for _, name := range sortedTSFieldNames(tsType) {
		if _, ok := properties[name]; !ok {
			t.Fatalf("%s field %s is missing from %s schema", tsType.Name, name, schemaName)
		}
	}
}

func assertTSOptionalityMatchesSchema(t *testing.T, tsType tsObjectType, schemaName string, schema map[string]any) {
	t.Helper()
	tsType = requireTSType(t, tsType, schemaName)
	required := schemaRequiredFields(schema)
	for _, name := range sortedTSFieldNames(tsType) {
		field := tsType.Fields[name]
		_, contractRequired := required[name]
		if field.Optional == contractRequired {
			t.Fatalf("%s field %s optional=%t, %s required=%t", tsType.Name, name, field.Optional, schemaName, contractRequired)
		}
	}
}

func requireTSType(t *testing.T, tsType tsObjectType, name string) tsObjectType {
	t.Helper()
	if tsType.Name == "" {
		t.Fatalf("missing TypeScript type %s", name)
	}
	return tsType
}

func schemaRequiredFields(schema map[string]any) map[string]bool {
	required := map[string]bool{}
	if values, ok := schema["required"].([]string); ok {
		for _, value := range values {
			required[value] = true
		}
	}
	return required
}

func queryParametersByName(parameters []apiParameter) map[string]apiParameter {
	result := map[string]apiParameter{}
	for _, parameter := range parameters {
		if parameter.In == "query" {
			result[parameter.Name] = parameter
		}
	}
	return result
}

func sortedTSFieldNames(tsType tsObjectType) []string {
	names := make([]string, 0, len(tsType.Fields))
	for name := range tsType.Fields {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedMapKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func assertStringSetsEqual(t *testing.T, label string, got []string, want []string) {
	t.Helper()
	if strings.Join(got, "\x00") != strings.Join(want, "\x00") {
		t.Fatalf("%s = %v, want %v", label, got, want)
	}
}
