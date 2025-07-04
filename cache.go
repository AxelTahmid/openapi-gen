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
}

// BuildTypeIndex scans the given roots and builds a type index for all Go types.
func BuildTypeIndex() *TypeIndex {
	idx := &TypeIndex{
		types:              make(map[string]map[string]*ast.TypeSpec),
		files:              make(map[string]*ast.File),
		externalKnownTypes: make(map[string]*Schema),
	}

	slog.Debug("[openapi] BuildTypeIndex: walking root")
	_ = filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil ||
			info.IsDir() ||
			!strings.HasSuffix(path, ".go") ||
			strings.HasSuffix(path, "_test.go") {
			return nil
		}

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			slog.Debug("[openapi] BuildTypeIndex: failed to parse file", "path", path, "err", err)
			return nil
		}
		idx.files[path] = file
		pkg := file.Name.Name
		if _, ok := idx.types[pkg]; !ok {
			idx.types[pkg] = make(map[string]*ast.TypeSpec)
		}
		for _, decl := range file.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						idx.types[pkg][ts.Name.Name] = ts
						slog.Debug(
							"[openapi] BuildTypeIndex: indexed type",
							"package",
							pkg,
							"type",
							ts.Name.Name,
							"file",
							path,
						)
					}
				}
			}
		}
		return nil
	})

	slog.Debug("[openapi] BuildTypeIndex: completed", "totalPackages", len(idx.types), "totalFiles", len(idx.files))
	return idx
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

// LookupUnqualifiedType searches for a type across all packages and returns the first match along with package name
func (idx *TypeIndex) LookupUnqualifiedType(typeName string) (*ast.TypeSpec, string) {
	if idx == nil {
		return nil, ""
	}
	for pkgName, pkgTypes := range idx.types {
		if typeSpec, exists := pkgTypes[typeName]; exists {
			return typeSpec, pkgName
		}
	}
	return nil, ""
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
