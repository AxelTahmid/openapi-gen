package openapi

import (
	"errors"
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

// TestInspectRoutes_NilRouter verifies InspectRoutes returns a RouteDiscoveryError when router is nil.
func TestInspectRoutes_NilRouter(t *testing.T) {
	routes, err := InspectRoutes(nil)
	if routes != nil {
		t.Errorf("Expected nil routes for nil router, got %v", routes)
	}
	var rdErr *RouteDiscoveryError
	if !errors.As(err, &rdErr) {
		t.Fatalf("Expected RouteDiscoveryError, got %T", err)
	}
	if rdErr.Operation != "inspect" {
		t.Errorf("Expected Operation=inspect, got %s", rdErr.Operation)
	}
}

// TestDiscoverRoutes_FiltersInternal ensures DiscoverRoutes filters swagger and openapi paths.
func TestDiscoverRoutes_FiltersInternal(t *testing.T) {
	r := chi.NewRouter()
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/swagger/doc", stub)
	r.Get("/openapi/data", stub)
	r.Get("/public", stub)
	routes, err := DiscoverRoutes(r)
	if err != nil {
		t.Fatalf("DiscoverRoutes returned error: %v", err)
	}
	if len(routes) != 1 || routes[0].Pattern != "/public" {
		t.Errorf("Expected only /public route, got %v", routes)
	}
}

// TestInspectRoutes_Middleware checks that middlewares are captured in RouteInfo.
func TestInspectRoutes_Middleware(t *testing.T) {
	r := chi.NewRouter()
	mw := func(next http.Handler) http.Handler { return next }
	r.Use(mw)
	stub := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	r.Get("/path", stub)
	routes, err := InspectRoutes(r)
	if err != nil {
		t.Fatalf("InspectRoutes returned error: %v", err)
	}
	found := false
	for _, ri := range routes {
		if ri.Pattern == "/path" {
			if len(ri.Middlewares) == 0 {
				t.Error("Expected middleware for route /path, got none")
			}
			found = true
		}
	}
	if !found {
		t.Error("Route /path not found in routes")
	}
}
