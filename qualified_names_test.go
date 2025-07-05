package openapi

import (
	"testing"
)

// Test types for qualified naming
type QualifiedTestUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type QualifiedTestOrder struct {
	ID   int               `json:"id"`
	User QualifiedTestUser `json:"user"`
}

// TestQualifiedNaming_Internal tests internal types get qualified names
func TestQualifiedNaming_Internal(t *testing.T) {
	gen := newTestSchemaGenerator()

	// Use an existing type from the openapi package
	schema := gen.GenerateSchema("Schema")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema reference for Schema")
	}

	// The reference should use qualified name
	expectedRef := "#/components/schemas/openapi.Schema"
	if schema.Ref != expectedRef {
		t.Errorf("expected ref %s, got %s", expectedRef, schema.Ref)
	}

	// Check that the schema is stored under the qualified name
	schemas := gen.GetSchemas()
	if _, exists := schemas["openapi.Schema"]; !exists {
		t.Error("schema should be stored under qualified name 'openapi.Schema'")
	}
}

// TestQualifiedNaming_External tests external types
func TestQualifiedNaming_External(t *testing.T) {
	gen := newTestSchemaGenerator()

	// Test with a known external type
	schema := gen.GenerateSchema("time.Time")
	if schema == nil {
		t.Fatal("expected schema for time.Time")
	}

	// Should be handled by external known types or as a basic mapping
	schemas := gen.GetSchemas()
	_, inSchemas := schemas["time.Time"]

	// Check external known types if not in regular schemas
	if !inSchemas && gen.typeIndex != nil {
		if extSchema, inExt := gen.typeIndex.externalKnownTypes["time.Time"]; inExt {
			if extSchema.Type != "string" || extSchema.Format != "date-time" {
				t.Error("time.Time should have proper external type mapping")
			}
		}
	}
}

// TestQualifiedNaming_NoDuplicates tests no duplicate schemas
func TestQualifiedNaming_NoDuplicates(t *testing.T) {
	gen := newTestSchemaGenerator()

	// Generate schema for same type multiple times using existing type
	schema1 := gen.GenerateSchema("Schema")
	schema2 := gen.GenerateSchema("Schema")
	schema3 := gen.GenerateSchema("openapi.Schema") // explicit qualified name

	// All should return the same reference
	if schema1.Ref != schema2.Ref || schema2.Ref != schema3.Ref {
		t.Error("multiple calls for same type should return same reference")
	}

	// Should only have one schema stored
	schemas := gen.GetSchemas()
	count := 0
	for name := range schemas {
		if name == "openapi.Schema" || name == "Schema" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 schema for Schema, got %d", count)
	}
}

// TestQualifiedNaming_Nested tests nested types use qualified names
func TestQualifiedNaming_Nested(t *testing.T) {
	gen := newTestSchemaGenerator()

	// Use existing nested types
	schema := gen.GenerateSchema("Components")
	if schema == nil || schema.Ref == "" {
		t.Fatal("expected schema reference for Components")
	}

	// Check that the Components schema exists under qualified name
	schemas := gen.GetSchemas()
	if _, exists := schemas["openapi.Components"]; !exists {
		t.Error("Components schema should exist under qualified name")
	}
}

// TestTypeIndexQualifiedLookup tests the new TypeIndex qualified lookup methods
func TestTypeIndexQualifiedLookup(t *testing.T) {
	resetTypeIndexForTesting()
	idx := BuildTypeIndex()

	t.Run("LookupQualifiedType works", func(t *testing.T) {
		ts := idx.LookupQualifiedType("openapi.Schema")
		if ts == nil {
			t.Error("should find Schema type by qualified name")
		}
	})

	t.Run("GetQualifiedTypeName works", func(t *testing.T) {
		qualifiedName := idx.GetQualifiedTypeName("Schema")
		if qualifiedName != "openapi.Schema" {
			t.Errorf("expected 'openapi.Schema', got '%s'", qualifiedName)
		}

		// Already qualified name should be returned as-is
		alreadyQualified := idx.GetQualifiedTypeName("sqlc.User")
		if alreadyQualified != "sqlc.User" {
			t.Errorf("expected 'sqlc.User', got '%s'", alreadyQualified)
		}
	})
}
