package openapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func CachedHandler(router chi.Router, cfg Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ensureTypeIndex()
		slog.Debug("[openapi] CachedHandler: checking cache validity")
		refresh := r.URL.Query().Get("refresh") == "true"

		cacheMutex.RLock()
		cachedSpec := specCache
		isValid := cacheValid && !refresh
		cacheMutex.RUnlock()

		if isValid {
			slog.Debug("[openapi] CachedHandler: cache hit, serving cached OpenAPI spec")
			writeSpec(w, cachedSpec)
			return
		}

		slog.Debug("[openapi] CachedHandler: cache miss or refresh requested, regenerating OpenAPI spec")
		generator := NewGeneratorWithCache(typeIndex)
		newSpec := generator.GenerateSpec(router, cfg)

		cacheMutex.Lock()
		specCache = newSpec
		cacheValid = true
		cacheMutex.Unlock()

		slog.Debug("[openapi] CachedHandler: cache updated, serving new OpenAPI spec")
		writeSpec(w, newSpec)
	}
}

func writeSpec(w http.ResponseWriter, spec Spec) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(spec); err != nil {
		http.Error(w, "Failed to encode OpenAPI spec", http.StatusInternalServerError)
	}
}

func InvalidateCache(w http.ResponseWriter, _ *http.Request) {
	cacheMutex.Lock()
	cacheValid = false
	cacheMutex.Unlock()
	slog.Debug("[openapi] InvalidateCache: OpenAPI cache invalidated")
	w.WriteHeader(http.StatusOK)
}

// GenerateOpenAPISpecFile generates the OpenAPI spec and writes it to the given file path.
func GenerateOpenAPISpecFile(router chi.Router, cfg Config, filePath string, refresh bool) error {
	ensureTypeIndex()
	slog.Debug("[openapi] GenerateOpenAPISpecFile: generating OpenAPI spec", "filePath", filePath)

	cacheMutex.RLock()
	cachedSpec := specCache
	isValid := cacheValid && !refresh
	cacheMutex.RUnlock()

	var spec Spec
	if isValid {
		slog.Debug("[openapi] GenerateOpenAPISpecFile: cache hit, using cached OpenAPI spec")
		spec = cachedSpec
	} else {
		slog.Debug("[openapi] GenerateOpenAPISpecFile: cache miss or refresh requested, regenerating OpenAPI spec")
		generator := NewGeneratorWithCache(typeIndex)
		spec = generator.GenerateSpec(router, cfg)

		cacheMutex.Lock()
		specCache = spec
		cacheValid = true
		cacheMutex.Unlock()

		slog.Debug("[openapi] GenerateOpenAPISpecFile: cache updated")
	}

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
		ensureTypeIndex()
		slog.Debug("[openapi] GenerateFileHandler: checking cache validity")
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
