package openapi

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestInspectRoutes(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Post("/bar/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	routes, err := InspectRoutes(r)
	if err != nil {
		t.Fatalf("InspectRoutes returned error: %v", err)
	}
	methods := make(map[string]bool)
	patterns := make(map[string]bool)
	for _, ri := range routes {
		if ri.HandlerName == "" {
			t.Errorf("Expected non-empty HandlerName for route %s", ri.Pattern)
		}
		methods[ri.Method] = true
		patterns[ri.Pattern] = true
	}
	if !methods["GET"] || !methods["POST"] {
		t.Errorf("Expected GET and POST in methods, got %v", methods)
	}
	if !patterns["/foo"] || !patterns["/bar/{id}"] {
		t.Errorf("Expected /foo and /bar/{id} in patterns, got %v", patterns)
	}
}

func TestDiscoverRoutes(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/foo", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	r.Get("/openapi.json", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	routes, err := DiscoverRoutes(r)
	if err != nil {
		t.Fatalf("DiscoverRoutes returned error: %v", err)
	}
	if len(routes) != 1 {
		t.Fatalf("Expected 1 route after filtering, got %d", len(routes))
	}
	if routes[0].Pattern != "/foo" {
		t.Errorf("Expected pattern /foo, got %s", routes[0].Pattern)
	}
}
