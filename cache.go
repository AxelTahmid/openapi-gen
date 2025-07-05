package openapi

import (
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	specCache     Spec
	cacheValid    bool
	cacheMutex    sync.RWMutex
	typeIndex     *TypeIndex
	typeIndexOnce sync.Once
)

// Find a way to add method that will add external known types to the type index
// This is useful for types that are not defined in the current package but are known to the OpenAPI spec,
// such as types from external libraries or standard library types that we want to document.
func ensureTypeIndex() {
	// debug.PrintStack()
	typeIndexOnce.Do(func() {
		slog.Debug("[openapi] cache.go: initializing typeIndex and externalKnownTypes")
		// Build type index once at startup
		typeIndex = BuildTypeIndex()

		slog.Debug("[openapi] cache.go: typeIndex built, setting externalKnownTypes")
		typeIndex.externalKnownTypes = map[string]*Schema{
			// JSON and raw data types
			"json.RawMessage": {Type: "object", Description: "Raw JSON data", AdditionalProperties: true},

			// PostgreSQL types
			"pgtype.Numeric":  {Type: "number", Description: "PostgreSQL numeric type"},
			"pgtype.Interval": {Type: "string", Description: "PostgreSQL interval type"},
			"pgtype.Timestamptz": {
				Type:        "string",
				Format:      "date-time",
				Description: "PostgreSQL timestamp with timezone",
			},
			"pgtype.Timestamp": {Type: "string", Format: "date-time", Description: "PostgreSQL timestamp"},
			"pgtype.UUID":      {Type: "string", Format: "uuid", Description: "PostgreSQL UUID type"},
			"pgtype.JSONB":     {Type: "object", Description: "PostgreSQL JSONB type", AdditionalProperties: true},
			"pgtype.JSON":      {Type: "object", Description: "PostgreSQL JSON type", AdditionalProperties: true},

			// Time types
			"time.Time": {Type: "string", Format: "date-time", Description: "RFC3339 date-time"},
			"*time.Time": {
				OneOf:       []*Schema{{Type: "string", Format: "date-time"}, {Type: "null"}},
				Description: "Nullable RFC3339 date-time",
			},
			"time.Duration": {Type: "string", Description: "Duration string (e.g., '1h30m')"},

			// UUID types
			"uuid.UUID": {Type: "string", Format: "uuid", Description: "UUID string"},
			"*uuid.UUID": {
				OneOf:       []*Schema{{Type: "string", Format: "uuid"}, {Type: "null"}},
				Description: "Nullable UUID string",
			},

			// Network types
			"net.IP":    {Type: "string", Format: "ipv4", Description: "IPv4 address"},
			"net.IPNet": {Type: "string", Description: "IP network (CIDR notation)"},
			"url.URL":   {Type: "string", Format: "uri", Description: "URL string"},
			"*url.URL": {
				OneOf:       []*Schema{{Type: "string", Format: "uri"}, {Type: "null"}},
				Description: "Nullable URL string",
			},

			// Database driver types
			"sql.NullString": {OneOf: []*Schema{{Type: "string"}, {Type: "null"}}, Description: "Nullable string"},
			"sql.NullInt64": {
				OneOf:       []*Schema{{Type: "integer", Format: "int64"}, {Type: "null"}},
				Description: "Nullable integer",
			},
			"sql.NullFloat64": {OneOf: []*Schema{{Type: "number"}, {Type: "null"}}, Description: "Nullable number"},
			"sql.NullBool":    {OneOf: []*Schema{{Type: "boolean"}, {Type: "null"}}, Description: "Nullable boolean"},
			"sql.NullTime": {
				OneOf:       []*Schema{{Type: "string", Format: "date-time"}, {Type: "null"}},
				Description: "Nullable date-time",
			},

			// Common Go types that might appear in APIs
			"big.Int": {Type: "string", Description: "Big integer as string"},
			"*big.Int": {
				OneOf:       []*Schema{{Type: "string"}, {Type: "null"}},
				Description: "Nullable big integer as string",
			},
			"decimal.Decimal": {Type: "string", Description: "Decimal number as string"},
			"*decimal.Decimal": {
				OneOf:       []*Schema{{Type: "string"}, {Type: "null"}},
				Description: "Nullable decimal number as string",
			},

			// Add more external types as needed
		}
		// Log the number of types and files indexed
		slog.Debug(
			"[openapi] cache.go: typeIndex initialized",
			"types",
			len(typeIndex.types),
			"files",
			len(typeIndex.files),
		)
	})
}

// TypeIndex provides fast lookup of type definitions by package and type name.
type TypeIndex struct {
	types              map[string]map[string]*ast.TypeSpec // package -> type -> spec
	files              map[string]*ast.File                // file path -> parsed file
	externalKnownTypes map[string]*Schema                  // external known types
	qualifiedTypes     map[string]*ast.TypeSpec            // qualified type name -> spec (e.g., "order.CreateReq")
	packageImports     map[string]string                   // import path -> package name (e.g., "github.com/user/sqlc" -> "sqlc")
}

// BuildTypeIndex scans the given roots and builds a type index for all Go types.
func BuildTypeIndex() *TypeIndex {
	idx := &TypeIndex{
		types:              make(map[string]map[string]*ast.TypeSpec),
		files:              make(map[string]*ast.File),
		externalKnownTypes: make(map[string]*Schema),
		qualifiedTypes:     make(map[string]*ast.TypeSpec),
		packageImports:     make(map[string]string),
	}

	// Find project root by looking for go.mod
	projectRoot := findProjectRoot()
	if projectRoot == "" {
		slog.Debug("[openapi] BuildTypeIndex: could not find project root, using current directory")
		projectRoot = "."
	} else {
		slog.Debug("[openapi] BuildTypeIndex: using project root", "root", projectRoot)
	}

	_ = filepath.Walk(projectRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil ||
			info.IsDir() ||
			!strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return err
		}

		return idx.indexFile(path)
	})

	slog.Debug("[openapi] BuildTypeIndex: completed", "totalPackages", len(idx.types), "totalFiles", len(idx.files))
	return idx
}

// indexFile processes a single Go file and indexes its types
func (idx *TypeIndex) indexFile(path string) error {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		slog.Debug("[openapi] BuildTypeIndex: failed to parse file", "path", path, "err", err)
		return nil // Continue with other files
	}

	idx.files[path] = file
	pkg := file.Name.Name

	if _, ok := idx.types[pkg]; !ok {
		idx.types[pkg] = make(map[string]*ast.TypeSpec)
	}

	// Index type declarations
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
			for _, spec := range gd.Specs {
				if ts, isTypeSpec := spec.(*ast.TypeSpec); isTypeSpec {
					typeName := ts.Name.Name
					qualifiedName := idx.getQualifiedTypeName(pkg, typeName)

					// Store in both maps
					idx.types[pkg][typeName] = ts
					idx.qualifiedTypes[qualifiedName] = ts

					slog.Debug(
						"[openapi] BuildTypeIndex: indexed type",
						"package", pkg,
						"type", typeName,
						"qualified", qualifiedName,
						"file", path,
					)
				}
			}
		}
	}

	return nil
}

func GetTypeIndex() *TypeIndex {
	if typeIndex == nil {
		slog.Error("[openapi] GetTypeIndex: typeIndex is nil, building type index")
		typeIndex = BuildTypeIndex()
	} else {
		slog.Debug("[openapi] GetTypeIndex: returning existing typeIndex")
	}
	return typeIndex
}

// LookupType returns the TypeSpec for a given package and type name, or nil if not found.
func (idx *TypeIndex) LookupType(pkg, typeName string) *ast.TypeSpec {
	if idx == nil {
		return nil
	}
	if pkgTypes, ok := idx.types[pkg]; ok {
		return pkgTypes[typeName]
	}
	return nil
}

// LookupQualifiedType returns the TypeSpec for a qualified type name (e.g., "order.CreateReq")
func (idx *TypeIndex) LookupQualifiedType(qualifiedName string) *ast.TypeSpec {
	if idx == nil {
		return nil
	}
	return idx.qualifiedTypes[qualifiedName]
}

// LookupUnqualifiedType searches for a type across all packages and returns the first match along with qualified name
func (idx *TypeIndex) LookupUnqualifiedType(typeName string) (*ast.TypeSpec, string) {
	if idx == nil {
		return nil, ""
	}

	// First check if it's a basic type
	if isBasicType(typeName) {
		return nil, ""
	}

	// Look for the type in all packages and return the qualified name
	for pkgName, pkgTypes := range idx.types {
		if typeSpec, exists := pkgTypes[typeName]; exists {
			qualifiedName := idx.getQualifiedTypeName(pkgName, typeName)
			return typeSpec, qualifiedName
		}
	}
	return nil, ""
}

// GetQualifiedTypeName returns the appropriate qualified name for a type
func (idx *TypeIndex) GetQualifiedTypeName(typeName string) string {
	// If already qualified, return as-is
	if strings.Contains(typeName, ".") {
		return typeName
	}

	// Look up the type and return its qualified name
	if _, qualifiedName := idx.LookupUnqualifiedType(typeName); qualifiedName != "" {
		return qualifiedName
	}

	// Fallback to original name
	return typeName
}

func AddExternalKnownType(name string, schema *Schema) {
	ensureTypeIndex() // Ensure typeIndex is initialized
	if typeIndex == nil {
		slog.Error("[openapi] AddExternalKnownType: typeIndex is nil, cannot add external type", "name", name)
		return
	}
	if typeIndex.externalKnownTypes == nil {
		typeIndex.externalKnownTypes = make(map[string]*Schema)
	}
	typeIndex.externalKnownTypes[name] = schema
	slog.Debug("[openapi] AddExternalKnownType: added external known type", "name", name)
}

// resetTypeIndexForTesting resets the type index for testing purposes
// This should only be used in tests
func resetTypeIndexForTesting() {
	typeIndex = nil
	typeIndexOnce = sync.Once{}
}

// getQualifiedTypeName creates a qualified type name for indexing.
// For external packages (like sqlc, pgtype), use the package name as-is.
// For internal project types, use package.TypeName format.
func (idx *TypeIndex) getQualifiedTypeName(pkg, typeName string) string {
	// Check if this is an external/third-party package
	if idx.isExternalPackage(pkg) {
		return pkg + "." + typeName
	}

	// For internal project types, use package.TypeName format
	return pkg + "." + typeName
}

// isExternalPackage determines if a package is external/third-party
func (idx *TypeIndex) isExternalPackage(pkg string) bool {
	// List of known external packages that should keep their qualified names
	externalPkgs := map[string]bool{
		"sqlc":    true,
		"pgtype":  true,
		"json":    true,
		"time":    true,
		"uuid":    true,
		"net":     true,
		"url":     true,
		"sql":     true,
		"big":     true,
		"decimal": true,
	}

	return externalPkgs[pkg]
}

// findProjectRoot finds the project root by looking for go.mod file
func findProjectRoot() string {
	// Start from current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Walk up the directory tree looking for go.mod
	for {
		goModPath := filepath.Join(currentDir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return currentDir
		}

		// Move up one directory
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Reached filesystem root
			break
		}
		currentDir = parentDir
	}

	return ""
}
