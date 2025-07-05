// Package openapi defines helpers for OpenAPI schema generation from Go types.
package openapi

import (
	"strings"
)

// isBasicType returns true if the Go type name denotes a primitive, array, pointer or map.
// This fast-path is used to decide whether to generate a basic or complex schema.
func isBasicType(typeName string) bool {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"string", "bool":
		return true
	}
	if strings.HasPrefix(typeName, "[]") || strings.HasPrefix(typeName, "*") || strings.HasPrefix(typeName, "map[") {
		return true
	}
	return false
}

// generateBasicTypeSchema returns a Schema for basic Go types (primitives, slices, pointers).
// It handles arrays and pointers by delegating to GenerateSchema for element types.
func (sg *SchemaGenerator) generateBasicTypeSchema(typeName string) *Schema {
	if strings.HasPrefix(typeName, "[]") {
		elem := strings.TrimPrefix(typeName, "[]")
		return &Schema{Type: "array", Items: sg.GenerateSchema(elem)}
	}
	if strings.HasPrefix(typeName, "*") {
		clean := strings.TrimPrefix(typeName, "*")
		return sg.GenerateSchema(clean)
	}
	// Fallback to mapping
	openapiType := mapGoTypeToOpenAPI(typeName)
	return &Schema{Type: openapiType, Description: "basic Go type"}
}

// mapGoTypeToOpenAPI maps a Go type name to the corresponding OpenAPI primitive type.
// Integer and unsigned integer kinds map to "integer", floats to "number", bool to "boolean", and string to "string".
// Other types default to "object".
func mapGoTypeToOpenAPI(typeName string) string {
	switch typeName {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	case "string":
		return "string"
	default:
		return "object"
	}
}
