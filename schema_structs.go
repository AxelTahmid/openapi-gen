// Package openapi provides struct-to-schema conversion logic.
package openapi

import (
	"go/ast"
	"log/slog"
	"strings"
)

// convertStructToSchema converts a Go AST struct type into an OpenAPI object schema.
func (sg *SchemaGenerator) convertStructToSchema(structType *ast.StructType) *Schema {
	slog.Debug("[openapi] convertStructToSchema: called")
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue // embedded field
		}

		fieldName := field.Names[0].Name
		if !ast.IsExported(fieldName) {
			continue // skip unexported
		}

		// Determine JSON property name
		jsonName := fieldName
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			if jsonTag := extractJSONTag(tag); jsonTag != "" && jsonTag != "-" {
				jsonName = jsonTag
			}
		}

		// Convert field type
		fieldSchema := sg.convertFieldType(field.Type)

		// Apply struct tag enhancements
		if field.Tag != nil {
			tag := strings.Trim(field.Tag.Value, "`")
			sg.applyEnhancedTags(fieldSchema, tag)
		}

		schema.Properties[jsonName] = fieldSchema

		// Ensure dependent schemas generated
		switch t := field.Type.(type) {
		case *ast.Ident:
			if t.Obj != nil && t.Obj.Kind == ast.Typ {
				qualified := sg.getQualifiedTypeName(t.Name)
				_ = sg.GenerateSchema(qualified)
			}
		case *ast.StarExpr:
			if ident, ok := t.X.(*ast.Ident); ok && ident.Obj != nil && ident.Obj.Kind == ast.Typ {
				qualified := sg.getQualifiedTypeName(ident.Name)
				_ = sg.GenerateSchema(qualified)
			}
		case *ast.SelectorExpr:
			if ident, ok := t.X.(*ast.Ident); ok {
				qualified := ident.Name + "." + t.Sel.Name
				_ = sg.GenerateSchema(qualified)
			}
		}

		// Determine required fields
		if !isPointerType(field.Type) && !hasOmitEmpty(field.Tag) {
			schema.Required = append(schema.Required, jsonName)
		}
	}

	return schema
}

// convertFieldType inspects a Go AST expression and returns its OpenAPI schema representation.
// It handles identifiers, pointers, arrays, selectors, maps, and empty interfaces.
func (sg *SchemaGenerator) convertFieldType(expr ast.Expr) *Schema {
	slog.Debug("[openapi] convertFieldType: called")

	switch t := expr.(type) {
	case *ast.Ident:
		// Basic Go types
		basic := mapGoTypeToOpenAPI(t.Name)
		if basic != "object" {
			return &Schema{Type: basic}
		}
		// Custom types
		qualified := sg.getQualifiedTypeName(t.Name)
		return sg.GenerateSchema(qualified)

	case *ast.StarExpr:
		// Pointer types: underlying schema
		return sg.convertFieldType(t.X)

	case *ast.ArrayType:
		// Arrays and slices
		elem := sg.convertFieldType(t.Elt)
		return &Schema{Type: "array", Items: elem}

	case *ast.SelectorExpr:
		// Qualified types (e.g., time.Time)
		if ident, ok := t.X.(*ast.Ident); ok {
			qualified := ident.Name + "." + t.Sel.Name
			return sg.GenerateSchema(qualified)
		}

	case *ast.MapType:
		// Maps as object with additionalProperties
		return &Schema{Type: "object", AdditionalProperties: sg.convertFieldType(t.Value)}

	case *ast.InterfaceType:
		// Empty interface as object
		return &Schema{Type: "object"}
	}

	slog.Debug("[openapi] convertFieldType: unknown type, defaulting to object")
	return &Schema{Type: "object"}
}

// isPointerType returns true if the given AST expression represents a pointer type.
func isPointerType(expr ast.Expr) bool {
	_, ok := expr.(*ast.StarExpr)
	return ok
}

// hasOmitEmpty reports whether the struct field tag includes the "omitempty" option.
func hasOmitEmpty(tag *ast.BasicLit) bool {
	if tag == nil {
		return false
	}
	return strings.Contains(tag.Value, "omitempty")
}
