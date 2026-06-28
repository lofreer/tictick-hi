package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

func TestAPIContractRouteExposesOpenAPIContract(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/api-contract", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	var contract openAPIContract
	if err := json.Unmarshal(recorder.Body.Bytes(), &contract); err != nil {
		t.Fatalf("decode contract: %v", err)
	}
	if contract.OpenAPI != "3.1.0" {
		t.Fatalf("openapi = %q, want 3.1.0", contract.OpenAPI)
	}
	if contract.Components.Schemas["APIErrorResponse"] == nil {
		t.Fatal("missing APIErrorResponse schema")
	}
	if contract.Paths["/api/system/api-contract"]["get"].Responses["200"].Content[jsonMediaType].Schema["type"] != "object" {
		t.Fatalf("contract route does not declare an object response schema")
	}
}

func TestAPIContractCoversCurrentFrontendRoutes(t *testing.T) {
	contract := apiContractDocument()
	expected := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/auth/login"},
		{http.MethodGet, "/api/auth/me"},
		{http.MethodPost, "/api/auth/logout"},
		{http.MethodGet, "/api/auth/sessions"},
		{http.MethodDelete, "/api/auth/sessions/{id}"},
		{http.MethodGet, "/api/data/tasks"},
		{http.MethodPost, "/api/data/tasks"},
		{http.MethodDelete, "/api/data/tasks/{id}"},
		{http.MethodPost, "/api/data/tasks/{id}/retry"},
		{http.MethodGet, "/api/data/tasks/{id}/gaps"},
		{http.MethodPost, "/api/data/tasks/{id}/repair-gaps"},
		{http.MethodPost, "/api/data/tasks/{id}/repair-gap"},
		{http.MethodPost, "/api/data/tasks/{id}/sync/{action}"},
		{http.MethodPost, "/api/data/tasks/{id}/realtime/{action}"},
		{http.MethodGet, "/api/candles"},
		{http.MethodGet, "/api/market/candle-gaps"},
		{http.MethodGet, "/api/market/instruments"},
		{http.MethodPost, "/api/market/instruments/sync"},
		{http.MethodGet, "/api/strategies"},
		{http.MethodGet, "/api/strategies/{id}"},
		{http.MethodGet, "/api/backtests"},
		{http.MethodPost, "/api/backtests"},
		{http.MethodGet, "/api/backtests/{id}"},
		{http.MethodGet, "/api/backtests/{id}/orders"},
		{http.MethodGet, "/api/backtests/{id}/intents"},
		{http.MethodGet, "/api/trading/tasks"},
		{http.MethodPost, "/api/trading/tasks"},
		{http.MethodGet, "/api/trading/tasks/{id}"},
		{http.MethodPost, "/api/trading/tasks/{id}/start"},
		{http.MethodPost, "/api/trading/tasks/{id}/pause"},
		{http.MethodPost, "/api/trading/tasks/{id}/stop"},
		{http.MethodGet, "/api/trading/tasks/{id}/intents"},
		{http.MethodGet, "/api/trading/tasks/{id}/orders"},
		{http.MethodGet, "/api/trading/tasks/{id}/executions"},
		{http.MethodGet, "/api/trading/tasks/{id}/positions"},
		{http.MethodGet, "/api/trading/tasks/{id}/notifications"},
		{http.MethodGet, "/api/system/notifications"},
		{http.MethodPost, "/api/system/notifications/{id}/retry"},
		{http.MethodGet, "/api/system/notifications/channels"},
		{http.MethodPost, "/api/system/notifications/channels"},
		{http.MethodGet, "/api/system/exchange-accounts"},
		{http.MethodPost, "/api/system/exchange-accounts"},
		{http.MethodGet, "/api/system/operators"},
		{http.MethodPost, "/api/system/operators"},
		{http.MethodPost, "/api/system/operators/{id}/enable"},
		{http.MethodPost, "/api/system/operators/{id}/disable"},
		{http.MethodGet, "/api/system/health"},
		{http.MethodGet, "/api/system/audit-events"},
		{http.MethodGet, "/api/system/api-contract"},
	}

	for _, route := range expected {
		method := strings.ToLower(route.method)
		if _, ok := contract.Paths[route.path][method]; !ok {
			t.Fatalf("contract missing %s %s", route.method, route.path)
		}
	}
}

func TestAPIContractDeclaresWriteSecurityAndErrorShape(t *testing.T) {
	operation := apiContractDocument().Paths["/api/system/exchange-accounts"]["post"]

	if !operationRequires(operation, "sessionCookie") || !operationRequires(operation, "csrfHeader") {
		t.Fatalf("write operation security = %#v, want session cookie and csrf header", operation.Security)
	}
	if ref := operation.RequestBody.Content[jsonMediaType].Schema["$ref"]; ref != "#/components/schemas/CreateExchangeAccount" {
		t.Fatalf("request body ref = %v, want CreateExchangeAccount", ref)
	}
	for _, status := range []string{"400", "401", "403", "405", "500"} {
		response := operation.Responses[status]
		if response.Content[jsonMediaType].Schema["$ref"] != "#/components/schemas/APIErrorResponse" {
			t.Fatalf("response %s schema = %#v, want APIErrorResponse ref", status, response.Content[jsonMediaType].Schema)
		}
	}
}

func TestAPIContractSchemasProtectSecretBoundary(t *testing.T) {
	schemas := apiContractDocument().Components.Schemas

	accountProperties := schemaProperties(t, schemas["ExchangeAccount"])
	if _, ok := accountProperties["apiKey"]; ok {
		t.Fatal("ExchangeAccount response schema exposes apiKey")
	}
	if _, ok := accountProperties["apiSecret"]; ok {
		t.Fatal("ExchangeAccount response schema exposes apiSecret")
	}
	if _, ok := accountProperties["credentialStatus"]; !ok {
		t.Fatal("ExchangeAccount response schema must expose credentialStatus")
	}

	createAccountProperties := schemaProperties(t, schemas["CreateExchangeAccount"])
	if _, ok := createAccountProperties["apiKey"]; !ok {
		t.Fatal("CreateExchangeAccount request schema missing apiKey")
	}
	if _, ok := createAccountProperties["apiSecret"]; !ok {
		t.Fatal("CreateExchangeAccount request schema missing apiSecret")
	}

	sessionProperties := schemaProperties(t, schemas["OperatorSession"])
	if _, ok := sessionProperties["tokenHash"]; ok {
		t.Fatal("OperatorSession response schema exposes tokenHash")
	}
}

func TestFrontendAPIRoutesAreCoveredByContract(t *testing.T) {
	routes, err := collectFrontendAPIRoutes(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) == 0 {
		t.Fatal("no frontend api routes found")
	}

	matchers := contractRouteMatchers(apiContractDocument())
	for _, route := range routes {
		operation, ok := matchers.match(route.Method, route.Path)
		if !ok {
			t.Fatalf("frontend route %s %s from %s is missing from API contract", route.Method, route.Path, route.File)
		}
		if unsafeMethod(route.Method) && operationRequires(operation, "sessionCookie") && !operationRequires(operation, "csrfHeader") {
			t.Fatalf("frontend write route %s %s from %s lacks csrfHeader contract", route.Method, route.Path, route.File)
		}
	}
}

type frontendAPIRoute struct {
	Method string
	Path   string
	File   string
}

type contractRouteMatcher struct {
	method    string
	path      string
	operation apiOperation
}

type contractRouteMatchersList []contractRouteMatcher

func collectFrontendAPIRoutes(root string) ([]frontendAPIRoute, error) {
	apiDir := filepath.Join(root, "web", "frontend", "src", "services", "api")
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		return nil, err
	}

	var routes []frontendAPIRoute
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".ts") || strings.HasSuffix(name, ".test.ts") || name == "client.ts" {
			continue
		}
		path := filepath.Join(apiDir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		for _, match := range frontendRouteExpression.FindAllSubmatch(content, -1) {
			routePath := string(match[2])
			if routePath == "" {
				routePath = string(match[3])
			}
			routes = append(routes, frontendAPIRoute{
				Method: strings.ToUpper(string(match[1])),
				Path:   normalizeFrontendPath(routePath),
				File:   filepath.ToSlash(filepath.Join("web", "frontend", "src", "services", "api", name)),
			})
		}
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].File != routes[j].File {
			return routes[i].File < routes[j].File
		}
		if routes[i].Path != routes[j].Path {
			return routes[i].Path < routes[j].Path
		}
		return routes[i].Method < routes[j].Method
	})
	return routes, nil
}

var frontendRouteExpression = regexp.MustCompile("(?s)apiClient\\.(get|post|delete)\\s*(?:<.*?>)?\\s*\\(\\s*(?:\"([^\"]+)\"|`([^`]+)`)")
var templateExpression = regexp.MustCompile(`\$\{[^}]+\}`)

func normalizeFrontendPath(path string) string {
	clean := templateExpression.ReplaceAllString(path, "{param}")
	if index := strings.Index(clean, "?"); index >= 0 {
		clean = clean[:index]
	}
	if strings.HasPrefix(clean, "/api/") {
		return clean
	}
	return "/api" + clean
}

func contractRouteMatchers(contract openAPIContract) contractRouteMatchersList {
	matchers := make(contractRouteMatchersList, 0)
	for path, item := range contract.Paths {
		for method, operation := range item {
			matchers = append(matchers, contractRouteMatcher{
				method:    strings.ToUpper(method),
				path:      path,
				operation: operation,
			})
		}
	}
	sort.Slice(matchers, func(i, j int) bool {
		if matchers[i].path != matchers[j].path {
			return matchers[i].path < matchers[j].path
		}
		return matchers[i].method < matchers[j].method
	})
	return matchers
}

func (matchers contractRouteMatchersList) match(method string, path string) (apiOperation, bool) {
	for _, matcher := range matchers {
		if matcher.method == method && pathsCompatible(matcher.path, path) {
			return matcher.operation, true
		}
	}
	return apiOperation{}, false
}

func pathsCompatible(contractPath string, frontendPath string) bool {
	contractParts := strings.Split(contractPath, "/")
	frontendParts := strings.Split(frontendPath, "/")
	if len(contractParts) != len(frontendParts) {
		return false
	}
	for index := range contractParts {
		if contractParts[index] == frontendParts[index] {
			continue
		}
		if pathParamSegment(contractParts[index]) || pathParamSegment(frontendParts[index]) {
			continue
		}
		return false
	}
	return true
}

func pathParamSegment(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func unsafeMethod(method string) bool {
	return method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve test file path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func operationRequires(operation apiOperation, name string) bool {
	for _, requirement := range operation.Security {
		if _, ok := requirement[name]; ok {
			return true
		}
	}
	return false
}

func schemaProperties(t *testing.T, schema map[string]any) map[string]any {
	t.Helper()
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema properties = %#v", schema["properties"])
	}
	return properties
}
