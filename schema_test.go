package openapi

import (
	"testing"
)

type Simple struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WithPointer struct {
	Name *string `json:"name,omitempty"`
}

type WithArray struct {
	Tags []string `json:"tags"`
}

type Nested struct {
	Simple Simple `json:"simple"`
}

type WithQualified struct {
	Other openapiTestOther `json:"other"`
}

type openapiTestOther struct {
	Foo int `json:"foo"`
}

func TestSchemaGenerator_SimpleStruct(t *testing.T) {
	gen := NewSchemaGenerator()
	schema := gen.GenerateSchema("Simple")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema ref for Simple")
	}
	all := gen.GetSchemas()
	if _, ok := all["Simple"]; !ok {
		t.Error("Simple schema not found in GetSchemas")
	}
}

func TestSchemaGenerator_PointerField(t *testing.T) {
	gen := NewSchemaGenerator()
	schema := gen.GenerateSchema("WithPointer")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema ref for WithPointer")
	}
}

func TestSchemaGenerator_ArrayField(t *testing.T) {
	gen := NewSchemaGenerator()
	schema := gen.GenerateSchema("WithArray")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema ref for WithArray")
	}
}

func TestSchemaGenerator_NestedStruct(t *testing.T) {
	gen := NewSchemaGenerator()
	schema := gen.GenerateSchema("Nested")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema ref for Nested")
	}
	all := gen.GetSchemas()
	if _, ok := all["Simple"]; !ok {
		t.Error("Nested should reference Simple schema")
	}
}

func TestSchemaGenerator_QualifiedType(t *testing.T) {
	gen := NewSchemaGenerator()
	schema := gen.GenerateSchema("openapiTestOther")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema ref for openapiTestOther")
	}
}

func TestSchemaGenerator_BasicTypes(t *testing.T) {
	gen := NewSchemaGenerator()
	cases := map[string]string{
		"int":     "integer",
		"string":  "string",
		"bool":    "boolean",
		"float64": "number",
		"[]int":   "array",
	}
	for goType, openapiType := range cases {
		schema := gen.GenerateSchema(goType)
		if schema.Type != openapiType {
			t.Errorf("expected %s for %s, got %s", openapiType, goType, schema.Type)
		}
	}
}

func Test_extractJSONTag(t *testing.T) {
	tag := "json:\"foo,omitempty\" xml:\"bar\""
	if got := extractJSONTag(tag); got != "foo" {
		t.Errorf("expected 'foo', got '%s'", got)
	}
}
