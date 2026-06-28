package api

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestAPIErrorCatalogHasUniqueKnownCodes(t *testing.T) {
	catalog := apiErrorCatalog()
	if len(catalog) == 0 {
		t.Fatal("api error catalog is empty")
	}

	seen := map[string]bool{}
	for _, definition := range catalog {
		if definition.Code == "" {
			t.Fatal("api error catalog contains an empty code")
		}
		if seen[definition.Code] {
			t.Fatalf("api error catalog contains duplicate code %q", definition.Code)
		}
		seen[definition.Code] = true
		if !apiErrorCodeKnown(apiErrorCode(definition.Code)) {
			t.Fatalf("catalog code %q is not known", definition.Code)
		}
		if definition.HTTPStatus < http.StatusBadRequest || definition.HTTPStatus > 599 {
			t.Fatalf("catalog code %q has invalid HTTP status %d", definition.Code, definition.HTTPStatus)
		}
		if strings.TrimSpace(definition.Description) == "" {
			t.Fatalf("catalog code %q has empty description", definition.Code)
		}
	}

	for _, code := range []apiErrorCode{
		apiErrorInvalidRequest,
		apiErrorUnauthorized,
		apiErrorForbidden,
		apiErrorCSRFRequired,
		apiErrorCSRFInvalid,
		apiErrorNotFound,
		apiErrorMethodNotAllowed,
		apiErrorConflict,
		apiErrorInvalidState,
		apiErrorMarketInstrumentNotActive,
		apiErrorDataSyncRetryRequiresFailed,
		apiErrorDataSyncCommandInvalidState,
		apiErrorTooManyRequests,
		apiErrorInternal,
		apiErrorRequestFailed,
	} {
		if !seen[string(code)] {
			t.Fatalf("catalog missing code %q", code)
		}
	}
}

func TestAPIErrorResponseSchemaUsesCatalogEnum(t *testing.T) {
	components := apiContractDocument().Components
	schemas := components.Schemas

	if len(components.ErrorCodes) != len(apiErrorCatalog()) {
		t.Fatalf("contract x-errorCodes length = %d, want %d", len(components.ErrorCodes), len(apiErrorCatalog()))
	}

	codeSchema := schemas["APIErrorCode"]
	if codeSchema["type"] != "string" {
		t.Fatalf("APIErrorCode schema type = %#v, want string", codeSchema["type"])
	}
	enum, ok := codeSchema["enum"].([]string)
	if !ok {
		t.Fatalf("APIErrorCode enum = %#v", codeSchema["enum"])
	}
	assertStringSetsEqual(t, "APIErrorCode enum", enum, apiErrorCodes())

	properties := schemaProperties(t, schemas["APIErrorResponse"])
	if properties["code"].(map[string]any)["$ref"] != "#/components/schemas/APIErrorCode" {
		t.Fatalf("APIErrorResponse code schema = %#v", properties["code"])
	}
}

func TestAPIContractErrorResponsesDeclareCatalogCodes(t *testing.T) {
	contract := apiContractDocument()
	catalog := apiErrorCatalogByCode()

	for path, item := range contract.Paths {
		for method, operation := range item {
			for statusText, response := range operation.Responses {
				status, err := strconv.Atoi(statusText)
				if err != nil || status < http.StatusBadRequest {
					continue
				}
				schema := response.Content[jsonMediaType].Schema
				if schema["$ref"] != "#/components/schemas/APIErrorResponse" {
					t.Fatalf("%s %s response %s schema = %#v, want APIErrorResponse", strings.ToUpper(method), path, statusText, schema)
				}
				if len(response.XErrorCodes) == 0 {
					t.Fatalf("%s %s response %s has no x-errorCodes", strings.ToUpper(method), path, statusText)
				}
				for _, code := range response.XErrorCodes {
					definition, ok := catalog[code]
					if !ok {
						t.Fatalf("%s %s response %s declares unknown error code %q", strings.ToUpper(method), path, statusText, code)
					}
					if definition.HTTPStatus != status {
						t.Fatalf("%s %s response %s declares code %q with catalog status %d", strings.ToUpper(method), path, statusText, code, definition.HTTPStatus)
					}
				}
			}
		}
	}
}

func TestWriteAPIErrorRejectsUnknownCode(t *testing.T) {
	recorder := httptest.NewRecorder()

	writeAPIError(recorder, http.StatusBadRequest, apiErrorCode("not_in_catalog"), "unsafe detail")

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusInternalServerError)
	}
	response := decodeAPIError(t, recorder)
	if response.Code != string(apiErrorInternal) || response.Message != "internal server error" {
		t.Fatalf("response = %#v, want safe internal error", response)
	}
}

func TestWriteAPIErrorCallsitesUseCatalogConstants(t *testing.T) {
	apiDir := filepath.Join(repoRoot(t), "internal", "web", "api")
	entries, err := os.ReadDir(apiDir)
	if err != nil {
		t.Fatal(err)
	}

	fileSet := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(apiDir, name)
		file, err := parser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 3 {
				return true
			}
			ident, ok := call.Fun.(*ast.Ident)
			if !ok || ident.Name != "writeAPIError" {
				return true
			}
			if literal, ok := call.Args[2].(*ast.BasicLit); ok && literal.Kind == token.STRING {
				position := fileSet.Position(literal.Pos())
				t.Fatalf("%s uses string literal API error code %s; use apiErrorCode catalog constants", position, literal.Value)
			}
			return true
		})
	}
}

func apiErrorCatalogByCode() map[string]apiErrorDefinition {
	catalog := map[string]apiErrorDefinition{}
	for _, definition := range apiErrorCatalog() {
		catalog[definition.Code] = definition
	}
	return catalog
}
