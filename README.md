# OpenAPI Go

A lightweight, annotation-driven OpenAPI 3.1 specification generator for Go HTTP services using Chi router.

## Features

-   **Chi Router Only**: Specifically designed for `go-chi/chi` router
-   **Top-Level Functions**: Handlers must be top-level functions (not methods) for annotation parsing
-   **Annotation-Driven**: Uses standard Swagger-style comments
-   **Dynamic Schema Generation**: Automatically generates JSON schemas from Go types
-   **Zero Configuration**: No manual type registration required
-   **Runtime Generation**: Builds OpenAPI spec dynamically from your routes

## Installation

```bash
go get github.com/yourusername/openapi-go
```

## Quick Start

### 1. Define Your Types

```go
// Request type
type CreateUserRequest struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   *int   `json:"age,omitempty"`
}

// Response type
type User struct {
    ID    int    `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
    Age   *int   `json:"age,omitempty"`
}
```

### 2. Create Top-Level Handler Function

**Important**: Handlers must be top-level functions, not struct methods, for annotation parsing to work.

```go
// CreateUser creates a new user
// @Summary Create a new user
// @Description Create a new user with the provided details
// @Tags users
// @Accept application/json
// @Produce application/json
// @Param user body CreateUserRequest true "User creation data"
// @Success 201 {object} User "User created successfully"
// @Failure 400 {object} ProblemDetails "Invalid request data"
// @Security BearerAuth
func CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    // ... your implementation here
}
```

### 3. Setup Router and OpenAPI Endpoint

```go
package main

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/yourusername/openapi-go"
)

func main() {
    r := chi.NewRouter()

    // Add your routes
    r.Post("/api/v1/users", CreateUser)

    // Configure and add OpenAPI endpoint
    config := openapi.Config{
        Title:   "My API",
        Version: "1.0.0",
        Server:  "http://localhost:8080",
    }
    r.Get("/openapi.json", openapi.Handler(r, config))

    http.ListenAndServe(":8080", r)
}
```

## Why Top-Level Functions?

This package uses Go's AST parsing to extract function comments and annotations. For the parser to correctly identify and process your handler functions, they must be defined as **top-level functions** rather than struct methods.

### ❌ Won't Work (Struct Methods)

```go
type UserHandler struct {
    // fields...
}

// This won't be found by the annotation parser
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    // @Summary Create user - THIS WON'T BE PARSED
}
```

### ✅ Will Work (Top-Level Functions)

```go
// This will be found and parsed correctly
// @Summary Create user
// @Description Create a new user with the provided details
func CreateUser(w http.ResponseWriter, r *http.Request) {
    // Implementation
}
```

## Supported Annotations

| Annotation     | Format                                                 | Description                   | Example                                                    |
| -------------- | ------------------------------------------------------ | ----------------------------- | ---------------------------------------------------------- |
| `@Summary`     | `@Summary <text>`                                      | Brief endpoint description    | `@Summary Create a new user`                               |
| `@Description` | `@Description <text>`                                  | Detailed endpoint description | `@Description Create a new user with the provided details` |
| `@Tags`        | `@Tags <tag1>,<tag2>`                                  | Comma-separated list of tags  | `@Tags users,management`                                   |
| `@Accept`      | `@Accept <media-type>`                                 | Request content type          | `@Accept application/json`                                 |
| `@Produce`     | `@Produce <media-type>`                                | Response content type         | `@Produce application/json`                                |
| `@Param`       | `@Param <name> <in> <type> <required> "<description>"` | Request parameters            | See examples below                                         |
| `@Success`     | `@Success <code> {<format>} <type> "<description>"`    | Success responses             | `@Success 200 {object} User "Success"`                     |
| `@Failure`     | `@Failure <code> {<format>} <type> "<description>"`    | Error responses               | `@Failure 400 {object} ProblemDetails "Bad Request"`       |
| `@Security`    | `@Security <scheme>`                                   | Security requirements         | `@Security BearerAuth`                                     |

### Parameter Types (`@Param`)

| Location | Example                                                | Description        |
| -------- | ------------------------------------------------------ | ------------------ |
| `body`   | `@Param user body CreateUserRequest true "User data"`  | Request body       |
| `path`   | `@Param id path int true "User ID"`                    | URL path parameter |
| `query`  | `@Param limit query int false "Page limit"`            | Query parameter    |
| `header` | `@Param Authorization header string true "Auth token"` | Header parameter   |

### Response Formats (`@Success` / `@Failure`)

| Format     | Example                                     | Description      |
| ---------- | ------------------------------------------- | ---------------- |
| `{object}` | `@Success 200 {object} User "Single user"`  | Single object    |
| `{array}`  | `@Success 200 {array} User "List of users"` | Array of objects |
| `{data}`   | `@Success 200 {data} string "Raw data"`     | Raw data type    |

## Schema Generation

The package automatically generates JSON schemas for your Go types:

### Features

-   **Automatic Discovery**: Finds types by scanning your project files
-   **Package-Aware**: Supports both local types (`User`) and package-qualified types (`db.User`)
-   **Struct Tag Support**: Respects `json` tags and `omitempty` directives
-   **Type Mapping**: Maps Go types to appropriate OpenAPI types
-   **Reference Resolution**: Handles circular references and type reuse

### Type Discovery Process

The schema generator searches for types in this order:

1. **Current Package**: For unqualified types like `CreateUserRequest`
2. **Project Packages**: Recursively searches under common directories (`internal/`, `pkg/`, etc.)
3. **Package-Qualified**: For types like `db.User` or `models.Product`

### Supported Type Patterns

| Pattern           | Example                     | Description                   |
| ----------------- | --------------------------- | ----------------------------- |
| Basic types       | `string`, `int`, `bool`     | Maps to OpenAPI primitives    |
| Structs           | `User`, `CreateUserRequest` | Generates object schemas      |
| Pointers          | `*User`, `*string`          | Makes fields optional         |
| Arrays/Slices     | `[]User`, `[]string`        | Generates array schemas       |
| Package-qualified | `db.User`, `models.Product` | Cross-package type references |

### Example Schema Generation

Given this Go struct:

```go
type User struct {
    ID        int       `json:"id"`
    Name      string    `json:"name"`
    Email     *string   `json:"email,omitempty"`
    Tags      []string  `json:"tags"`
    CreatedAt time.Time `json:"created_at"`
}
```

The generator produces this OpenAPI schema:

```json
{
    "type": "object",
    "properties": {
        "id": { "type": "integer" },
        "name": { "type": "string" },
        "email": { "type": "string" },
        "tags": {
            "type": "array",
            "items": { "type": "string" }
        },
        "created_at": { "type": "string" }
    },
    "required": ["id", "name", "tags", "created_at"]
}
```

## Configuration

### Basic Configuration

```go
config := openapi.Config{
    Title:       "My API",
    Description: "API description",
    Version:     "1.0.0",
    Server:      "https://api.example.com",
}
```

### Full Configuration

```go
config := openapi.Config{
    Title:          "User Management API",
    Description:    "Complete user management system",
    Version:        "2.1.0",
    TermsOfService: "https://example.com/terms",
    Server:         "https://api.example.com",
    Contact: &openapi.Contact{
        Name:  "API Support Team",
        Email: "api-support@example.com",
        URL:   "https://example.com/support",
    },
    License: &openapi.License{
        Name: "Apache 2.0",
        URL:  "https://www.apache.org/licenses/LICENSE-2.0.html",
    },
}
```

## Integration Examples

### With Authentication Middleware

```go
// Protected routes will automatically include BearerAuth requirement
r.Route("/api/v1", func(r chi.Router) {
    // Public routes
    r.Post("/auth/login", LoginUser)
    r.Post("/auth/register", RegisterUser)

    // Protected routes (will require BearerAuth)
    r.Group(func(r chi.Router) {
        r.Use(authMiddleware) // JWT middleware
        r.Get("/users", ListUsers)
        r.Post("/users", CreateUser)
        r.Get("/users/{id}", GetUser)
    })
})
```

### Multiple API Versions

```go
// v1 routes
r.Route("/api/v1", func(r chi.Router) {
    r.Get("/users", V1ListUsers)
    r.Post("/users", V1CreateUser)
})

// v2 routes
r.Route("/api/v2", func(r chi.Router) {
    r.Get("/users", V2ListUsers)
    r.Post("/users", V2CreateUser)
})

// Separate OpenAPI specs for each version
r.Get("/api/v1/openapi.json", openapi.Handler(r, v1Config))
r.Get("/api/v2/openapi.json", openapi.Handler(r, v2Config))
```

### Error Handling Best Practices

```go
// Define standard error response type
type ProblemDetails struct {
    Type     string                 `json:"type"`
    Title    string                 `json:"title"`
    Status   int                    `json:"status"`
    Detail   string                 `json:"detail,omitempty"`
    Instance string                 `json:"instance,omitempty"`
    Extra    map[string]interface{} `json:"-"`
}

// Use consistent error responses in annotations
// @Failure 400 {object} ProblemDetails "Bad Request"
// @Failure 401 {object} ProblemDetails "Unauthorized"
// @Failure 404 {object} ProblemDetails "Not Found"
// @Failure 500 {object} ProblemDetails "Internal Server Error"
```

## Common Issues & Solutions

### Issue: Annotations Not Being Parsed

**Problem**: Your handler annotations aren't showing up in the generated OpenAPI spec.

**Solution**: Ensure your handlers are top-level functions:

```go
// ❌ This won't work (method)
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {}

// ✅ This will work (top-level function)
func CreateUser(w http.ResponseWriter, r *http.Request) {}
```

### Issue: Types Not Found

**Problem**: Referenced types in `@Param` or `@Success` annotations return generic schemas.

**Solutions**:

1. Ensure the type is defined in your project
2. Use fully qualified names for types in other packages: `db.User` instead of `User`
3. Check that the file containing the type is parseable Go code

### Issue: Missing Required Fields

**Problem**: Generated schema doesn't mark fields as required correctly.

**Solution**: The generator marks fields as required if they:

-   Are not pointer types (`string` vs `*string`)
-   Don't have `omitempty` in their JSON tag
-   Are exported (capitalized)

```go
type User struct {
    ID    int     `json:"id"`              // Required
    Name  string  `json:"name"`            // Required
    Email *string `json:"email,omitempty"` // Optional
}
```

## Testing Your OpenAPI Spec

### Validate Generated Spec

```bash
# Test the endpoint
curl http://localhost:8080/openapi.json | jq .

# Validate with swagger-codegen
swagger-codegen validate -i http://localhost:8080/openapi.json
```

### Integration with Swagger UI

```go
import "github.com/swaggo/http-swagger"

// Add Swagger UI endpoint
r.Get("/docs/*", httpSwagger.Handler(
    httpSwagger.URL("http://localhost:8080/openapi.json"),
))
```

## Architecture

The package consists of 4 main components:

| Component            | File             | Purpose                                       |
| -------------------- | ---------------- | --------------------------------------------- |
| **Types**            | `types.go`       | OpenAPI 3.1 type definitions                  |
| **Annotations**      | `annotations.go` | Comment parsing and annotation extraction     |
| **Schema Generator** | `schema.go`      | Dynamic Go type to JSON schema conversion     |
| **Generator**        | `generator.go`   | Main specification generator and HTTP handler |

## Limitations

-   **Chi Router Only**: Only works with `go-chi/chi` router
-   **Top-Level Functions**: Handler methods on structs are not supported
-   **Comment Parsing**: Requires properly formatted Go comments
-   **File Access**: Needs access to source files for type discovery

## Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

-   **Issues**: Report bugs and feature requests on GitHub
-   **Discussions**: Join discussions about usage and best practices
-   **Documentation**: Check the examples and test files for more usage patterns
