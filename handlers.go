package openapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

// CachedHandler returns an HTTP handler that serves the OpenAPI specification.
// The specification is cached and only regenerated when refresh=true is passed
// as a query parameter or when the cache is invalidated.
func CachedHandler(router chi.Router, cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refresh := r.URL.Query().Get("refresh") == "true"
		spec := fetchSpec(router, cfg, refresh)
		writeSpec(w, spec)
	}
}

// writeSpec writes the OpenAPI specification as JSON to the response writer.
// Sets appropriate content type and handles encoding errors gracefully.
func writeSpec(w http.ResponseWriter, spec Spec) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		slog.Error("[openapi] writeSpec: failed to encode JSON", "error", err)
		http.Error(w, "Failed to encode OpenAPI spec", http.StatusInternalServerError)
	}
}

// InvalidateCache invalidates the cached OpenAPI specification.
// The next request will trigger regeneration of the specification.
func InvalidateCache(w http.ResponseWriter, _ *http.Request) {
	cacheMutex.Lock()
	cacheValid = false
	cacheMutex.Unlock()
	slog.Debug("[openapi] InvalidateCache: OpenAPI cache invalidated")
	w.WriteHeader(http.StatusOK)
}

// GenerateOpenAPISpecFile generates the OpenAPI spec and writes it to the given file path.
func GenerateOpenAPISpecFile(router chi.Router, cfg Config, filePath string, refresh bool) error {
	slog.Debug("[openapi] GenerateOpenAPISpecFile: generating OpenAPI spec", "filePath", filePath)

	spec := fetchSpec(router, cfg, refresh)

	slog.Debug("[openapi] GenerateOpenAPISpecFile: writing OpenAPI spec to file", "version", spec.Info.Version)

	file, err := os.Create(filePath)
	if err != nil {
		slog.Debug("[openapi] GenerateOpenAPISpecFile: failed to create file", "err", err)
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err = enc.Encode(spec); err != nil {
		slog.Debug("[openapi] GenerateOpenAPISpecFile: failed to write file", "err", err)
		return err
	}

	slog.Debug("[openapi] GenerateOpenAPISpecFile: openapi.json written successfully")
	return nil
}

// GenerateFileHandler is an HTTP handler that generates the OpenAPI spec file and returns a status message.
func GenerateFileHandler(router chi.Router, cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		refresh := r.URL.Query().Get("refresh") == "true"

		err := GenerateOpenAPISpecFile(router, cfg, "openapi.json", refresh)
		if err != nil {
			http.Error(w, "Failed to write file", http.StatusInternalServerError)
			return
		}

		slog.Debug("[openapi] GenerateFileHandler: openapi.json written successfully")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"message":"openapi.json created"}`))
	}
}

// getCachedSpec retrieves the current cached spec and whether it is still valid.
func getCachedSpec(refresh bool) (Spec, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()
	return specCache, cacheValid && !refresh
}

// setCachedSpec updates the cache with a new spec and marks it valid.
func setCachedSpec(s Spec) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()
	specCache = s
	cacheValid = true
}

// fetchSpec handles cache: returns cached spec or regenerates if needed.
func fetchSpec(router chi.Router, cfg Config, refresh bool) Spec {
	ensureTypeIndex()
	if spec, ok := getCachedSpec(refresh); ok {
		return spec
	}
	gen := NewGeneratorWithCache(typeIndex)
	newSpec := gen.GenerateSpec(router, cfg)
	setCachedSpec(newSpec)
	return newSpec
}
