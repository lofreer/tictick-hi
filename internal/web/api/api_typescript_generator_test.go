package api

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFrontendAPIGeneratedTypesAreCurrent(t *testing.T) {
	path := generatedAPITypesPath(t)
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	expected := mustGenerateFrontendAPITypes(t)
	if string(current) != expected {
		t.Fatalf("%s is stale; run scripts/generate-api-types.sh", filepath.ToSlash(path))
	}
}

func TestWriteGeneratedFrontendAPITypes(t *testing.T) {
	if os.Getenv("TICTICK_WRITE_GENERATED_API_TYPES") != "1" {
		t.Skip("set TICTICK_WRITE_GENERATED_API_TYPES=1 to rewrite generated API types")
	}
	path := generatedAPITypesPath(t)
	if err := os.WriteFile(path, []byte(mustGenerateFrontendAPITypes(t)), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustGenerateFrontendAPITypes(t *testing.T) string {
	t.Helper()
	content, err := generatedFrontendAPITypescript(apiContractDocument())
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func generatedAPITypesPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "web", "frontend", "src", "types", "api.generated.ts")
}
