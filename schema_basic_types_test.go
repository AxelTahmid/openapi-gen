package openapi

import (
	"testing"
)

func TestIsBasicType(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"int", true},
		{"string", true},
		{"bool", true},
		{"[]int", true},
		{"*string", true},
		{"map[string]int", true},
		{"MyStruct", false},
		{"[]MyStruct", true},
		{"map[MyStruct]string", true},
		{"custom.Type", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isBasicType(tc.name)
			AssertEqual(t, tc.want, got)
		})
	}
}

func TestMapGoTypeToOpenAPI(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{"int", "integer"},
		{"uint32", "integer"},
		{"float64", "number"},
		{"bool", "boolean"},
		{"string", "string"},
		{"MyStruct", "object"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mapGoTypeToOpenAPI(tc.name)
			AssertEqual(t, tc.want, got)
		})
	}
}

func TestGenerateBasicTypeSchema(t *testing.T) {
	sg := NewTestSchemaGenerator()
	tests := []struct {
		name string
		want *Schema
	}{
		{"int", &Schema{Type: "integer", Description: "basic Go type"}},
		{"[]string", &Schema{Type: "array", Items: &Schema{Type: "string", Description: "basic Go type"}}},
		{"*bool", &Schema{Type: "boolean", Description: "basic Go type"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sg.generateBasicTypeSchema(tc.name)
			AssertDeepEqual(t, tc.want, got)
		})
	}
}
