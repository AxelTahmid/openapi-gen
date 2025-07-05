package openapi

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

// TestGenerateSpecRoutes ensures that GenerateSpec includes discovered routes and parameters.
func TestGenerateSpecRoutes(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/foo/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	cfg := Config{Title: "Test Service", Version: "1.2.3"}
	g := NewGenerator()
	spec := g.GenerateSpec(r, cfg)

	// Check Info
	if spec.Info.Title != cfg.Title {
		t.Errorf("expected Info.Title %q, got %q", cfg.Title, spec.Info.Title)
	}
	if spec.Info.Version != cfg.Version {
		t.Errorf("expected Info.Version %q, got %q", cfg.Version, spec.Info.Version)
	}

	// Check path presence and operation
	paths := spec.Paths
	if _, ok := paths["/foo/{id}"]; !ok {
		t.Fatalf("expected path '/foo/{id}' in spec.Paths")
	}
	ops := paths["/foo/{id}"]
	op, ok := ops["get"]
	if !ok {
		t.Fatalf("expected GET operation for '/foo/{id}'")
	}

	// Verify path parameter id
	if len(op.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(op.Parameters))
	}
	p := op.Parameters[0]
	if p.Name != "id" || p.In != "path" || !p.Required {
		t.Errorf("unexpected path parameter: %+v", p)
	}
}
