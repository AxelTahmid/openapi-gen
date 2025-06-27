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
		slog.Info("[openapi] cache.go: initializing typeIndex and externalKnownTypes")
		// Build type index once at startup
		typeIndex = BuildTypeIndex()

		slog.Info("[openapi] cache.go: typeIndex built, setting externalKnownTypes")
		typeIndex.externalKnownTypes = map[string]*Schema{
			"json.RawMessage":    {Type: "object", Description: "raw json byte slice, used for dynamic JSON data"},
			"pgtype.Numeric":     {Type: "number", Description: "external type: postgres numeric"},
			"pgtype.Interval":    {Type: "string", Description: "external type: postgres interval"},
			"pgtype.Timestamptz": {Type: "date-time", Description: "external type: postgres timezone aware timestamp"},
			"pgtype.Timestamp":   {Type: "date-time", Description: "external type: postgres timestamp"},
			"pgtype.UUID":        {Type: "string", Description: "external type: postgres uuid type"},
			"pgtype.JSONB":       {Type: "object", Description: "external type: postgres JSONB"},
			"pgtype.JSON":        {Type: "object", Description: "external type: postgres JSON"},
			// Add more external types as needed
		}
		// Log the number of types and files indexed
		slog.Info(
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
		types: make(map[string]map[string]*ast.TypeSpec),
		files: make(map[string]*ast.File),
	}

	slog.Info("[openapi] BuildTypeIndex: walking root")
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
			slog.Info("[openapi] BuildTypeIndex: failed to parse file", "path", path, "err", err)
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
						slog.Info(
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

	slog.Info("[openapi] BuildTypeIndex: completed", "totalPackages", len(idx.types), "totalFiles", len(idx.files))
	return idx
}

func GetTypeIndex() *TypeIndex {
	if typeIndex == nil {
		slog.Error("[openapi] GetTypeIndex: typeIndex is nil, building type index")
		typeIndex = BuildTypeIndex()
	} else {
		slog.Info("[openapi] GetTypeIndex: returning existing typeIndex")
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
	if typeIndex == nil {
		slog.Error("[openapi] AddExternalKnownType: typeIndex is nil, cannot add external type", "name", name)
		return
	}
	typeIndex.externalKnownTypes[name] = schema
	slog.Info("[openapi] AddExternalKnownType: added external known type", "name", name)
}
