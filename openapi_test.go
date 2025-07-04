package openapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Test types - simple structs for testing
type TestUser struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
}

type TestCreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type TestErrorResponse struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// Test handler with annotations
// @Summary Create a new user
// @Description Creates a new user with the provided information
// @Tags users
// @Accept application/json
// @Produce application/json
// @Param user body TestCreateUserRequest true "User creation data"
// @Success 201 {object} TestUser "User created successfully"
// @Failure 400 {object} TestErrorResponse "Invalid request"
func CreateUserHandler(w http.ResponseWriter, r *http.Request) {
	email := "test@example.com"
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(TestUser{ID: 1, Name: "Test", Email: &email})
}

// @Summary Get all users
// @Description Retrieve a list of all users
// @Tags users
// @Produce application/json
// @Success 200 {array} TestUser "List of users"
func GetUsersHandler(w http.ResponseWriter, r *http.Request) {
	email := "test@example.com"
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode([]TestUser{{ID: 1, Name: "Test", Email: &email}})
}

// @Summary Get user by ID
// @Description Retrieve a specific user by their ID
// @Tags users
// @Produce application/json
// @Param id path int true "User ID"
// @Success 200 {object} TestUser "User found"
// @Failure 404 {object} TestErrorResponse "User not found"
func GetUserByIDHandler(w http.ResponseWriter, r *http.Request) {
	email := "test@example.com"
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(TestUser{ID: 1, Name: "Test", Email: &email})
}

// TestSchemaGeneration tests the core schema generation functionality
func TestSchemaGeneration(t *testing.T) {
	gen := NewSchemaGenerator()

	tests := []struct {
		name     string
		typeName string
		wantType string
	}{
		{"Basic string", "string", "string"},
		{"Basic int", "int", "integer"},
		{"Basic bool", "bool", "boolean"},
		{"Basic float", "float64", "number"},
		{"Array type", "[]string", "array"},
		{"Pointer type", "*string", "string"},
		{"Map type", "map[string]interface{}", "object"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := gen.GenerateSchema(tt.typeName)
			if schema == nil {
				t.Fatalf("GenerateSchema returned nil for %s", tt.typeName)
			}

			if schema.Type != tt.wantType {
				t.Errorf("Expected type %s for %s, got %s", tt.wantType, tt.typeName, schema.Type)
			}

			// Array types should have items
			if tt.wantType == "array" && schema.Items == nil {
				t.Errorf("Array schema should have items defined")
			}
		})
	}
}

// TestAnnotationParsing tests annotation parsing functionality
func TestAnnotationParsing(t *testing.T) {
	annotation := ParseAnnotations("openapi_test.go", "CreateUserHandler")
	if annotation == nil {
		t.Fatal("ParseAnnotations returned nil")
	}

	if annotation.Summary != "Create a new user" {
		t.Errorf("Expected summary 'Create a new user', got '%s'", annotation.Summary)
	}

	if len(annotation.Tags) != 1 || annotation.Tags[0] != "users" {
		t.Errorf("Expected tags [users], got %v", annotation.Tags)
	}

	if len(annotation.Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(annotation.Parameters))
	}

	if annotation.Success == nil || annotation.Success.DataType != "TestUser" {
		t.Errorf("Expected success response with TestUser type")
	}

	if len(annotation.Failures) != 1 || annotation.Failures[0].StatusCode != 400 {
		t.Errorf("Expected one failure with status 400")
	}
}

// TestSpecGeneration tests the main spec generation functionality
func TestSpecGeneration(t *testing.T) {
	cfg := Config{
		Title:       "Test API",
		Version:     "1.0.0",
		Description: "Test API Description",
	}

	r := chi.NewRouter()
	r.Post("/users", CreateUserHandler)
	r.Get("/users", GetUsersHandler)
	r.Get("/users/{id}", GetUserByIDHandler)

	gen := NewGenerator()
	spec := gen.GenerateSpec(r, cfg)

	// Check basic spec structure
	if spec.OpenAPI != "3.1.0" {
		t.Errorf("Expected OpenAPI version 3.1.0, got %s", spec.OpenAPI)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Info.Title)
	}

	if spec.Info.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", spec.Info.Version)
	}

	// Check paths exist
	expectedPaths := []string{"/users", "/users/{id}"}
	for _, path := range expectedPaths {
		if _, exists := spec.Paths[path]; !exists {
			t.Errorf("Expected path %s to exist in spec", path)
		}
	}

	// Check operations exist
	usersPath := spec.Paths["/users"]
	if _, exists := usersPath["post"]; !exists {
		t.Error("Expected POST operation on /users")
	}
	if _, exists := usersPath["get"]; !exists {
		t.Error("Expected GET operation on /users")
	}

	userByIdPath := spec.Paths["/users/{id}"]
	if _, exists := userByIdPath["get"]; !exists {
		t.Error("Expected GET operation on /users/{id}")
	}

	// Check components exist
	if spec.Components == nil {
		t.Error("Expected components to be defined")
	}
}

// TestCachedHandler tests the HTTP handler functionality
func TestCachedHandler(t *testing.T) {
	cfg := Config{
		Title:   "Test API",
		Version: "1.0.0",
	}

	r := chi.NewRouter()
	r.Get("/users", GetUsersHandler)

	handler := CachedHandler(r, cfg)

	// Test normal request
	req := httptest.NewRequest("GET", "/openapi.json", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Verify response is valid JSON
	var spec Spec
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Errorf("Response is not valid JSON: %v", err)
	}

	if spec.Info.Title != "Test API" {
		t.Errorf("Expected title 'Test API', got '%s'", spec.Info.Title)
	}
}

// TestTypeIndex tests the type indexing functionality
func TestTypeIndex(t *testing.T) {
	// Reset type index for clean test
	resetTypeIndexForTesting()

	idx := BuildTypeIndex()
	if idx == nil {
		t.Fatal("BuildTypeIndex returned nil")
	}

	if idx.types == nil {
		t.Error("TypeIndex.types should not be nil")
	}

	if idx.files == nil {
		t.Error("TypeIndex.files should not be nil")
	}

	if idx.externalKnownTypes == nil {
		t.Error("TypeIndex.externalKnownTypes should not be nil")
	}

	// Test lookup functionality
	// Look for a type that should exist in the openapi package
	spec, pkg := idx.LookupUnqualifiedType("Spec")
	if spec == nil {
		t.Error("Should find Spec type in openapi package")
	}
	if pkg != "openapi" {
		t.Errorf("Expected package 'openapi', got '%s'", pkg)
	}
}

// TestExternalTypes tests external type handling
func TestExternalTypes(t *testing.T) {
	resetTypeIndexForTesting()
	ensureTypeIndex()

	// Add external type
	AddExternalKnownType("CustomType", &Schema{
		Type:        "string",
		Description: "Custom external type",
	})

	gen := NewSchemaGenerator()
	schema := gen.GenerateSchema("CustomType")

	if schema == nil {
		t.Fatal("GenerateSchema returned nil for external type")
	}

	if schema.Type != "string" {
		t.Errorf("Expected type 'string', got '%s'", schema.Type)
	}

	if schema.Description != "Custom external type" {
		t.Errorf("Expected description 'Custom external type', got '%s'", schema.Description)
	}
}

// TestInvalidateCache tests cache invalidation
func TestInvalidateCache(t *testing.T) {
	req := httptest.NewRequest("POST", "/invalidate", nil)
	w := httptest.NewRecorder()

	InvalidateCache(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestConfigValidation tests configuration validation
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		valid  bool
	}{
		{
			name: "Valid minimal config",
			config: Config{
				Title:   "Test API",
				Version: "1.0.0",
			},
			valid: true,
		},
		{
			name: "Valid full config",
			config: Config{
				Title:       "Test API",
				Version:     "1.0.0",
				Description: "Test Description",
				Contact: &Contact{
					Name:  "Test Contact",
					Email: "test@example.com",
				},
				License: &License{
					Name: "MIT",
					URL:  "https://opensource.org/licenses/MIT",
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := chi.NewRouter()
			gen := NewGenerator()
			spec := gen.GenerateSpec(r, tt.config)

			if spec.Info.Title != tt.config.Title {
				t.Errorf("Title not set correctly")
			}

			if spec.Info.Version != tt.config.Version {
				t.Errorf("Version not set correctly")
			}

			if tt.config.Contact != nil && spec.Info.Contact == nil {
				t.Error("Contact should be set")
			}

			if tt.config.License != nil && spec.Info.License == nil {
				t.Error("License should be set")
			}
		})
	}
}
