package openapi

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RouteInfo holds metadata about each registered route
// including HTTP method, path pattern, handler name, and function.
type RouteInfo struct {
	Method      string
	Pattern     string
	HandlerName string
	HandlerFunc http.HandlerFunc
	Middlewares []func(http.Handler) http.Handler
}

// InspectRoutes walks a Chi router and returns a list of RouteInfo.
// Returns an error if the router traversal fails.
func InspectRoutes(r chi.Router) ([]RouteInfo, error) {
	var routes []RouteInfo
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		// Attempt to extract http.HandlerFunc
		var hf http.HandlerFunc
		switch h := handler.(type) {
		case http.HandlerFunc:
			hf = h
		default:
			// wrap other handlers
			hf = h.ServeHTTP
		}
		name := runtime.FuncForPC(reflect.ValueOf(hf).Pointer()).Name()
		routes = append(routes, RouteInfo{
			Method:      method,
			Pattern:     route,
			HandlerName: name,
			HandlerFunc: hf,
			Middlewares: middlewares,
		})
		return nil
	})
	return routes, err
}

// DiscoverRoutes filters out internal OpenAPI routes and returns usable RouteInfo.
// DiscoverRoutes returns only non-internal routes for OpenAPI spec assembly.
func DiscoverRoutes(r chi.Router) ([]RouteInfo, error) {
	// Retrieve all routes via InspectRoutes
	infos, err := InspectRoutes(r)
	if err != nil {
		return nil, err
	}
	var filtered []RouteInfo
	for _, ri := range infos {
		// Skip OpenAPI internals
		if strings.Contains(ri.Pattern, "/swagger") || strings.Contains(ri.Pattern, "/openapi") {
			continue
		}
		filtered = append(filtered, ri)
	}
	return filtered, nil
}
