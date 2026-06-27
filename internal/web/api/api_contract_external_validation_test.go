package api

import (
	"context"
	"net/http"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestAPIContractValidatesWithExternalOpenAPIValidator(t *testing.T) {
	_, server, auth := newAuthenticatedTestServer(t)

	recorder := serveAuthenticated(server, auth, http.MethodGet, "/api/system/api-contract", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = false
	document, err := loader.LoadFromData(recorder.Body.Bytes())
	if err != nil {
		t.Fatalf("external OpenAPI loader rejected /api/system/api-contract: %v", err)
	}
	if err := document.Validate(context.Background()); err != nil {
		t.Fatalf("external OpenAPI validator rejected /api/system/api-contract: %v", err)
	}
}
