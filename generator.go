package openapi

import (
	"log/slog"
	"net/http"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
)

// Generator creates OpenAPI specifications from Chi routers.
type Generator struct {
	schemaGen *SchemaGenerator
}

type Config struct {
	Title          string
	Description    string
	Version        string
	TermsOfService string
	Server         string
	Contact        *Contact
	License        *License
}

type Contact struct {
	Name  string
	URL   string
	Email string
}

type License struct {
	Name string
	URL  string
}

type Spec struct {
	OpenAPI           string                 `json:"openapi"`
	Info              Info                   `json:"info"`
	JSONSchemaDialect string                 `json:"jsonSchemaDialect,omitempty"` // OpenAPI 3.1 feature
	Servers           []Server               `json:"servers,omitempty"`
	Paths             map[string]PathItem    `json:"paths"`
	Webhooks          Webhooks               `json:"webhooks,omitempty"` // OpenAPI 3.1 feature
	Components        *Components            `json:"components,omitempty"`
	Tags              []Tag                  `json:"tags,omitempty"`
	Security          []SecurityRequirement  `json:"security,omitempty"`
	ExternalDocs      *ExternalDocumentation `json:"externalDocs,omitempty"`
}

type Info struct {
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	TermsOfService string   `json:"termsOfService,omitempty"`
	Contact        *Contact `json:"contact,omitempty"`
	License        *License `json:"license,omitempty"`
	Version        string   `json:"version"`
}

type Server struct {
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
}

type PathItem map[string]Operation

type Operation struct {
	Tags         []string               `json:"tags,omitempty"`
	Summary      string                 `json:"summary,omitempty"`
	Description  string                 `json:"description,omitempty"`
	ExternalDocs *ExternalDocumentation `json:"externalDocs,omitempty"`
	OperationID  string                 `json:"operationId,omitempty"`
	Parameters   []Parameter            `json:"parameters,omitempty"`
	RequestBody  *RequestBody           `json:"requestBody,omitempty"`
	Responses    map[string]Response    `json:"responses"`
	Callbacks    map[string]Callback    `json:"callbacks,omitempty"`
	Deprecated   bool                   `json:"deprecated,omitempty"`
	Security     []SecurityRequirement  `json:"security,omitempty"`
	Servers      []Server               `json:"servers,omitempty"`
}

type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"`
	Description string  `json:"description,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

type RequestBody struct {
	Description string                     `json:"description,omitempty"`
	Content     map[string]MediaTypeObject `json:"content"`
	Required    bool                       `json:"required,omitempty"`
}

type MediaTypeObject struct {
	Schema   *Schema             `json:"schema,omitempty"`
	Example  interface{}         `json:"example,omitempty"`
	Examples map[string]Example  `json:"examples,omitempty"`
	Encoding map[string]Encoding `json:"encoding,omitempty"`
}

type Response struct {
	Description string                     `json:"description"`
	Headers     map[string]Header          `json:"headers,omitempty"`
	Content     map[string]MediaTypeObject `json:"content,omitempty"`
	Links       map[string]Link            `json:"links,omitempty"`
}

type Schema struct {
	// Basic type information
	Type                 string             `json:"type,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Required             []string           `json:"required,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	Description          string             `json:"description,omitempty"`

	// JSON Schema Draft 2020-12 compliance
	Format      string              `json:"format,omitempty"`
	Pattern     string              `json:"pattern,omitempty"`
	Minimum     *float64            `json:"minimum,omitempty"`
	Maximum     *float64            `json:"maximum,omitempty"`
	MinLength   *int                `json:"minLength,omitempty"`
	MaxLength   *int                `json:"maxLength,omitempty"`
	MinItems    *int                `json:"minItems,omitempty"`
	MaxItems    *int                `json:"maxItems,omitempty"`
	UniqueItems *bool               `json:"uniqueItems,omitempty"`
	Enum        []interface{}       `json:"enum,omitempty"`
	Const       interface{}         `json:"const,omitempty"`
	Default     interface{}         `json:"default,omitempty"`
	Example     interface{}         `json:"example,omitempty"`
	Examples    map[string]*Example `json:"examples,omitempty"`

	// OpenAPI 3.1 composition
	OneOf []*Schema `json:"oneOf,omitempty"`
	AnyOf []*Schema `json:"anyOf,omitempty"`
	AllOf []*Schema `json:"allOf,omitempty"`
	Not   *Schema   `json:"not,omitempty"`

	// Metadata
	Title         string                 `json:"title,omitempty"`
	Deprecated    *bool                  `json:"deprecated,omitempty"`
	ReadOnly      *bool                  `json:"readOnly,omitempty"`
	WriteOnly     *bool                  `json:"writeOnly,omitempty"`
	XML           *XML                   `json:"xml,omitempty"`
	ExternalDocs  *ExternalDocumentation `json:"externalDocs,omitempty"`
	Discriminator *Discriminator         `json:"discriminator,omitempty"`
}

type Components struct {
	Schemas         map[string]Schema         `json:"schemas,omitempty"`
	Responses       map[string]Response       `json:"responses,omitempty"`
	Parameters      map[string]Parameter      `json:"parameters,omitempty"`
	Examples        map[string]Example        `json:"examples,omitempty"`
	RequestBodies   map[string]RequestBody    `json:"requestBodies,omitempty"`
	Headers         map[string]Header         `json:"headers,omitempty"`
	SecuritySchemes map[string]SecurityScheme `json:"securitySchemes,omitempty"`
	Links           map[string]Link           `json:"links,omitempty"`
	Callbacks       map[string]Callback       `json:"callbacks,omitempty"`
	PathItems       map[string]PathItem       `json:"pathItems,omitempty"` // OpenAPI 3.1 feature
}

type Example struct {
	Summary       string      `json:"summary,omitempty"`
	Description   string      `json:"description,omitempty"`
	Value         interface{} `json:"value,omitempty"`
	ExternalValue string      `json:"externalValue,omitempty"`
}

// XML represents OpenAPI 3.1 XML metadata
type XML struct {
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Prefix    string `json:"prefix,omitempty"`
	Attribute bool   `json:"attribute,omitempty"`
	Wrapped   bool   `json:"wrapped,omitempty"`
}

// ExternalDocumentation represents OpenAPI 3.1 external documentation
type ExternalDocumentation struct {
	Description string `json:"description,omitempty"`
	URL         string `json:"url"`
}

// Discriminator represents OpenAPI 3.1 discriminator for polymorphic schemas
type Discriminator struct {
	PropertyName string            `json:"propertyName"`
	Mapping      map[string]string `json:"mapping,omitempty"`
}

// Header represents OpenAPI 3.1 header object
type Header struct {
	Description     string              `json:"description,omitempty"`
	Required        bool                `json:"required,omitempty"`
	Deprecated      bool                `json:"deprecated,omitempty"`
	AllowEmptyValue bool                `json:"allowEmptyValue,omitempty"`
	Schema          *Schema             `json:"schema,omitempty"`
	Example         interface{}         `json:"example,omitempty"`
	Examples        map[string]*Example `json:"examples,omitempty"`
}

// Link represents OpenAPI 3.1 link object for describing relationships between operations
type Link struct {
	OperationRef string                 `json:"operationRef,omitempty"`
	OperationId  string                 `json:"operationId,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	RequestBody  interface{}            `json:"requestBody,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Server       *Server                `json:"server,omitempty"`
}

// Callback represents OpenAPI 3.1 callback object
type Callback map[string]*PathItem

// Encoding represents OpenAPI 3.1 encoding for request/response content
type Encoding struct {
	ContentType   string             `json:"contentType,omitempty"`
	Headers       map[string]*Header `json:"headers,omitempty"`
	Style         string             `json:"style,omitempty"`
	Explode       *bool              `json:"explode,omitempty"`
	AllowReserved bool               `json:"allowReserved,omitempty"`
}

// Webhooks represents OpenAPI 3.1 webhooks - a new feature in OpenAPI 3.1
type Webhooks map[string]*PathItem

// SecurityRequirement represents a security requirement
type SecurityRequirement map[string][]string

type SecurityScheme struct {
	Type         string `json:"type"`
	Scheme       string `json:"scheme,omitempty"`
	BearerFormat string `json:"bearerFormat,omitempty"`
	Description  string `json:"description,omitempty"`
}

type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func NewGeneratorWithCache(typeIndex *TypeIndex) *Generator {
	return &Generator{
		schemaGen: &SchemaGenerator{
			schemas:   make(map[string]*Schema),
			typeIndex: typeIndex,
		},
	}
}

// NewGenerator creates a Generator with a default TypeIndex.
func NewGenerator() *Generator {
	ensureTypeIndex()
	return NewGeneratorWithCache(typeIndex)
}

// GenerateSpec creates an OpenAPI 3.1 specification from a Chi router.
func (g *Generator) GenerateSpec(router chi.Router, cfg Config) Spec {
	slog.Debug("[openapi] GenerateSpec: called", "title", cfg.Title, "version", cfg.Version)
	spec := Spec{
		OpenAPI:           "3.1.0",
		JSONSchemaDialect: "https://spec.openapis.org/oas/3.1/dialect/base",
		Info: Info{
			Title:          cfg.Title,
			Version:        cfg.Version,
			Description:    cfg.Description,
			TermsOfService: cfg.TermsOfService,
			Contact:        cfg.Contact,
			License:        cfg.License,
		},
		Paths: make(map[string]PathItem),
		Components: &Components{
			Schemas:         make(map[string]Schema),
			SecuritySchemes: make(map[string]SecurityScheme),
			Responses:       make(map[string]Response),
			Parameters:      make(map[string]Parameter),
			Examples:        make(map[string]Example),
			RequestBodies:   make(map[string]RequestBody),
			Headers:         make(map[string]Header),
			Links:           make(map[string]Link),
			Callbacks:       make(map[string]Callback),
			PathItems:       make(map[string]PathItem),
		},
	}

	// Add server if configured
	if cfg.Server != "" {
		slog.Debug("[openapi] GenerateSpec: adding server", "server", cfg.Server)
		spec.Servers = []Server{{URL: cfg.Server, Description: "API Server"}}
	}

	slog.Debug("[openapi] GenerateSpec: adding security scheme")
	// Add standard security scheme
	spec.Components.SecuritySchemes["BearerAuth"] = SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
		Description:  "JWT token authentication",
	}

	// Add standard schemas
	g.addStandardSchemas(&spec)

	// Discover routes via DiscoverRoutes
	tags := make(map[string]bool)
	routes, err := DiscoverRoutes(router)
	if err != nil {
		slog.Warn("[openapi] GenerateSpec: InspectRoutes error", "error", err)
	}
	for _, ri := range routes {
		method := ri.Method
		route := ri.Pattern
		handler := ri.HandlerFunc
		slog.Debug("[openapi] GenerateSpec: processing route", "method", method, "route", route)
		pathKey := convertRouteToOpenAPIPath(route)
		operation := g.buildOperation(handler, route, method, ri.Middlewares)

		if spec.Paths[pathKey] == nil {
			spec.Paths[pathKey] = make(PathItem)
		}
		spec.Paths[pathKey][strings.ToLower(method)] = operation
		for _, tag := range operation.Tags {
			tags[tag] = true
		}
	}

	slog.Debug("[openapi] GenerateSpec: building tags array")
	// Build tags array
	spec.Tags = g.buildTags(tags)

	// Add generated schemas with qualified names
	for name, schema := range g.schemaGen.GetSchemas() {
		// Ensure the schema key is qualified
		qualifiedName := name
		slog.Debug(
			"[openapi] GenerateSpec: processing schema",
			"name",
			name,
			"hasQualifier",
			strings.Contains(name, "."),
		)
		if !strings.Contains(name, ".") && g.schemaGen.typeIndex != nil {
			if qualified := g.schemaGen.typeIndex.GetQualifiedTypeName(name); qualified != name {
				qualifiedName = qualified
				slog.Debug(
					"[openapi] GenerateSpec: qualifying schema key",
					"original",
					name,
					"qualified",
					qualifiedName,
				)
			}
		}
		spec.Components.Schemas[qualifiedName] = schema
	}

	slog.Debug("[openapi] GenerateSpec: completed", "path_count", len(spec.Paths))
	return spec
}

// buildOperation creates an OpenAPI operation from a handler.
func (g *Generator) buildOperation(
	handler http.Handler,
	route, method string,
	middlewares []func(http.Handler) http.Handler,
) Operation {
	slog.Debug("[openapi] buildOperation: called", "route", route, "method", method)
	// Get handler info
	handlerInfo := g.extractHandlerInfo(handler)

	// Parse annotations if handler info is available
	var annotations *Annotation
	if handlerInfo != nil && handlerInfo.File != "" {
		slog.Debug(
			"buildOperation: parsing annotations",
			"file",
			handlerInfo.File,
			"function",
			handlerInfo.FunctionName,
		)
		var err error
		annotations, err = ParseAnnotations(handlerInfo.File, handlerInfo.FunctionName)
		if err != nil {
			slog.Warn("[openapi] buildOperation: annotations parse error", "error", err)
		}
	}

	// Build operation
	operation := Operation{
		OperationID: generateOperationID(method, route),
		Parameters:  []Parameter{}, // Start with empty parameters, will add from route and annotations
		Responses:   g.buildResponses(annotations),
	}

	// Add path parameters from route
	pathParams := extractPathParameters(route)
	operation.Parameters = append(operation.Parameters, pathParams...)

	// Set summary and description
	if annotations != nil {
		operation.Summary = annotations.Summary
		operation.Description = annotations.Description
		operation.Tags = annotations.Tags

		// Convert and add parameters from annotations
		for _, param := range annotations.Parameters {
			// Skip body parameters - they should be handled as request body, not parameters
			if param.In == "body" {
				continue
			}

			// For non-body parameters (query, header, path), use simple type mapping
			operation.Parameters = append(operation.Parameters, Parameter{
				Name:        param.Name,
				In:          param.In,
				Description: param.Description,
				Required:    param.Required,
				Schema:      &Schema{Type: mapGoTypeToOpenAPI(param.Type)},
			})
		}
	}

	// Add default tag if none specified
	if len(operation.Tags) == 0 {
		operation.Tags = []string{extractResourceFromRoute(route)}
	}

	// Add request body for POST/PUT/PATCH
	if method == "POST" || method == "PUT" || method == "PATCH" {
		operation.RequestBody = g.buildRequestBody(annotations)
	}

	// Determine security requirements
	if hasJWTMiddleware(middlewares) {
		operation.Security = []SecurityRequirement{{"BearerAuth": {}}}
	}

	slog.Debug("[openapi] buildOperation: completed", "operationId", operation.OperationID)
	return operation
}

type HandlerInfo struct {
	File         string
	FunctionName string
	Package      string
}

// extractHandlerInfo gets information about a handler function.
func (g *Generator) extractHandlerInfo(handler http.Handler) *HandlerInfo {
	slog.Debug("[openapi] extractHandlerInfo: called")
	handlerValue := reflect.ValueOf(handler)
	if handlerValue.Kind() != reflect.Func {
		return nil
	}

	pc := handlerValue.Pointer()
	funcInfo := runtime.FuncForPC(pc)
	if funcInfo == nil {
		return nil
	}

	file, _ := funcInfo.FileLine(pc)
	name := funcInfo.Name()

	// Extract function name from full name
	if lastDot := strings.LastIndex(name, "."); lastDot != -1 {
		name = name[lastDot+1:]
		// Remove -fm suffix if present
		name = strings.TrimSuffix(name, "-fm")
	}

	slog.Debug("[openapi] extractHandlerInfo: found handler info", "file", file, "function", name)
	return &HandlerInfo{
		File:         file,
		FunctionName: name,
	}
}

// buildResponses creates response definitions.
func (g *Generator) buildResponses(annotations *Annotation) map[string]Response {
	slog.Debug("[openapi] buildResponses: called")
	responses := make(map[string]Response)

	// Add success response
	if annotations != nil && annotations.Success != nil {
		statusCode := strconv.Itoa(annotations.Success.StatusCode)
		responses[statusCode] = Response{
			Description: annotations.Success.Description,
			Content: map[string]MediaTypeObject{
				"application/json": {
					Schema: g.generateResponseSchema(annotations.Success.DataType),
				},
			},
		}
	} else {
		responses["200"] = Response{
			Description: "Successful response",
			Content: map[string]MediaTypeObject{
				"application/json": {
					Schema: &Schema{Type: "object"},
				},
			},
		}
	}

	// Add error responses from annotations
	if annotations != nil {
		for _, failure := range annotations.Failures {
			statusCode := strconv.Itoa(failure.StatusCode)
			responses[statusCode] = Response{
				Description: failure.Description,
				Content: map[string]MediaTypeObject{
					"application/problem+json": {
						Schema: &Schema{Ref: "#/components/schemas/ProblemDetails"},
					},
				},
			}
		}
	}

	// Add standard error responses if not present
	standardErrors := map[string]Response{
		"400": {
			Description: "Bad Request",
			Content: map[string]MediaTypeObject{
				"application/problem+json": {
					Schema: &Schema{Ref: "#/components/schemas/ProblemDetails"},
				},
			},
		},
		"401": {
			Description: "Unauthorized",
			Content: map[string]MediaTypeObject{
				"application/problem+json": {
					Schema: &Schema{Ref: "#/components/schemas/ProblemDetails"},
				},
			},
		},
		"500": {
			Description: "Internal Server Error",
			Content: map[string]MediaTypeObject{
				"application/problem+json": {
					Schema: &Schema{Ref: "#/components/schemas/ProblemDetails"},
				},
			},
		},
	}

	for code, response := range standardErrors {
		if _, exists := responses[code]; !exists {
			responses[code] = response
		}
	}

	slog.Debug("[openapi] buildResponses: completed", "response_count", len(responses))
	return responses
}

// buildRequestBody creates request body definition.
func (g *Generator) buildRequestBody(annotations *Annotation) *RequestBody {
	slog.Debug("[openapi] buildRequestBody: called")
	var schema *Schema
	description := "Request body"

	// Try to get from annotations first
	if annotations != nil {
		for _, param := range annotations.Parameters {
			if param.In == "body" {
				slog.Debug("[openapi] buildRequestBody: found body parameter", "type", param.Type)
				// Generate proper schema for the request body type
				schema = g.schemaGen.GenerateSchema(param.Type)
				if param.Description != "" {
					description = param.Description
				}
				break
			}
		}
	}

	// Default schema if no annotation provided
	if schema == nil {
		slog.Debug("[openapi] buildRequestBody: no body parameter found, using default object schema")
		schema = &Schema{Type: "object"}
	}

	return &RequestBody{
		Description: description,
		Required:    true,
		Content: map[string]MediaTypeObject{
			"application/json": {Schema: schema},
		},
	}
}

// generateResponseSchema creates a response schema.
func (g *Generator) generateResponseSchema(dataType string) *Schema {
	slog.Debug("[openapi] generateResponseSchema: called", "dataType", dataType)
	if dataType == "" {
		return &Schema{Type: "object"}
	}

	// Handle array types
	if strings.HasPrefix(dataType, "[]") {
		itemType := strings.TrimPrefix(dataType, "[]")
		return &Schema{
			Type:  "array",
			Items: g.schemaGen.GenerateSchema(itemType),
		}
	}

	// Handle pointer types
	if strings.HasPrefix(dataType, "*") {
		cleanType := strings.TrimPrefix(dataType, "*")
		return g.schemaGen.GenerateSchema(cleanType)
	}

	return g.schemaGen.GenerateSchema(dataType)
}

// addStandardSchemas adds predefined schemas.
func (g *Generator) addStandardSchemas(spec *Spec) {
	slog.Debug("[openapi] addStandardSchemas: adding ProblemDetails schema")
	spec.Components.Schemas["ProblemDetails"] = Schema{
		Type: "object",
		Properties: map[string]*Schema{
			"type":   {Type: "string", Description: "A URI reference identifying the problem type"},
			"title":  {Type: "string", Description: "A short, human-readable summary of the problem"},
			"status": {Type: "integer", Description: "The HTTP status code"},
			"detail": {Type: "string", Description: "Detailed explanation of the problem"},
			"instance": {
				Type:        "string",
				Description: "A URI reference identifying the specific instance of the problem",
			},
		},
		Required: []string{"type", "title", "status"},
	}
}

// buildTags creates tags array from collected tag names.
func (g *Generator) buildTags(tagNames map[string]bool) []Tag {
	slog.Debug("[openapi] buildTags: called", "tag_count", len(tagNames))
	var tags []Tag
	for name := range tagNames {
		tags = append(tags, Tag{
			Name:        name,
			Description: capitalize(name) + " related operations",
		})
	}

	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })

	return tags
}

// OpenAPI 3.1 Helper Functions

// AddWebhook adds a webhook to the specification
func (g *Generator) AddWebhook(spec *Spec, name string, pathItem PathItem) {
	if spec.Webhooks == nil {
		spec.Webhooks = make(Webhooks)
	}
	spec.Webhooks[name] = &pathItem
}

// CreateOneOfSchema creates a oneOf schema for polymorphic types
func CreateOneOfSchema(schemas ...*Schema) *Schema {
	return &Schema{
		OneOf: schemas,
	}
}

// CreateAnyOfSchema creates an anyOf schema for union types
func CreateAnyOfSchema(schemas ...*Schema) *Schema {
	return &Schema{
		AnyOf: schemas,
	}
}

// CreateAllOfSchema creates an allOf schema for composition
func CreateAllOfSchema(schemas ...*Schema) *Schema {
	return &Schema{
		AllOf: schemas,
	}
}

// AddSchemaExample adds an example to a schema
func AddSchemaExample(schema *Schema, name string, example Example) {
	if schema.Examples == nil {
		schema.Examples = make(map[string]*Example)
	}
	schema.Examples[name] = &example
}

// AddResponseHeader adds a header to a response
func AddResponseHeader(response *Response, name string, header Header) {
	if response.Headers == nil {
		response.Headers = make(map[string]Header)
	}
	response.Headers[name] = header
}

// AddResponseLink adds a link to a response
func AddResponseLink(response *Response, name string, link Link) {
	if response.Links == nil {
		response.Links = make(map[string]Link)
	}
	response.Links[name] = link
}

// SetSchemaFormat sets the format for a schema (e.g., "date-time", "email", "uuid")
func SetSchemaFormat(schema *Schema, format string) {
	schema.Format = format
}

// SetSchemaPattern sets a regex pattern for string validation
func SetSchemaPattern(schema *Schema, pattern string) {
	schema.Pattern = pattern
}

// SetSchemaRange sets minimum and maximum values for numeric types
func SetSchemaRange(schema *Schema, min, max *float64) {
	schema.Minimum = min
	schema.Maximum = max
}

// SetSchemaStringLength sets minimum and maximum length for strings
func SetSchemaStringLength(schema *Schema, minLen, maxLen *int) {
	schema.MinLength = minLen
	schema.MaxLength = maxLen
}

// SetSchemaArrayConstraints sets array constraints
func SetSchemaArrayConstraints(schema *Schema, minItems, maxItems *int, uniqueItems *bool) {
	schema.MinItems = minItems
	schema.MaxItems = maxItems
	schema.UniqueItems = uniqueItems
}

// AddSchemaEnum adds enum values to a schema
func AddSchemaEnum(schema *Schema, values ...interface{}) {
	schema.Enum = append(schema.Enum, values...)
}

// MarkSchemaDeprecated marks a schema as deprecated
func MarkSchemaDeprecated(schema *Schema) {
	deprecated := true
	schema.Deprecated = &deprecated
}

// MarkSchemaReadOnly marks a schema as read-only
func MarkSchemaReadOnly(schema *Schema) {
	readOnly := true
	schema.ReadOnly = &readOnly
}

// MarkSchemaWriteOnly marks a schema as write-only
func MarkSchemaWriteOnly(schema *Schema) {
	writeOnly := true
	schema.WriteOnly = &writeOnly
}

// Helper functions for OpenAPI generation

// convertRouteToOpenAPIPath converts Chi route to OpenAPI path format.
func convertRouteToOpenAPIPath(route string) string {
	// Chi uses {param} format, which is the same as OpenAPI
	return route
}

// extractPathParameters extracts path parameters from route.
func extractPathParameters(route string) []Parameter {
	var params []Parameter
	parts := strings.Split(route, "/")

	for _, part := range parts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			paramName := strings.Trim(part, "{}")
			params = append(params, Parameter{
				Name:     paramName,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		}
	}

	return params
}

// generateOperationID creates an operation ID from method and route.
func generateOperationID(method, route string) string {
	parts := strings.Split(strings.Trim(route, "/"), "/")
	var cleanParts []string

	for _, part := range parts {
		if part != "" && !strings.Contains(part, "{") {
			cleanParts = append(cleanParts, capitalize(part))
		}
	}

	return strings.ToLower(method) + strings.Join(cleanParts, "")
}

// extractResourceFromRoute extracts resource name from route.
func extractResourceFromRoute(route string) string {
	parts := strings.Split(strings.Trim(route, "/"), "/")

	// Skip common prefixes
	for _, part := range parts {
		if part != "" && part != "api" && part != "v1" && !strings.Contains(part, "{") {
			return part
		}
	}

	return "default"
}

// hasJWTMiddleware checks if JWT middleware is present.
func hasJWTMiddleware(middlewares []func(http.Handler) http.Handler) bool {
	for _, mw := range middlewares {
		funcName := runtime.FuncForPC(reflect.ValueOf(mw).Pointer()).Name()
		if strings.Contains(funcName, "jwt") ||
			strings.Contains(funcName, "JWT") ||
			strings.Contains(funcName, "auth") {
			return true
		}
	}
	return false
}

// capitalize returns the string with its first rune uppercased.
func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, size := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[size:]
}
