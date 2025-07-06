package openapi

import (
	"go/ast"
	"testing"
)

func TestIsConstantOfType(t *testing.T) {
	sg := NewTestSchemaGenerator()
	// ValueSpec without Type
	vs1 := &ast.ValueSpec{}
	AssertEqual(t, false, sg.isConstantOfType(vs1, "MyType"))
	// ValueSpec with matching Ident type
	vs2 := &ast.ValueSpec{Type: &ast.Ident{Name: "MyType"}}
	AssertEqual(t, true, sg.isConstantOfType(vs2, "MyType"))
	// ValueSpec with non-matching type
	vs3 := &ast.ValueSpec{Type: &ast.Ident{Name: "Other"}}
	AssertEqual(t, false, sg.isConstantOfType(vs3, "MyType"))
}

func TestHandleEnumType_NoEnum(t *testing.T) {
	sg := NewTestSchemaGenerator()
	// No type index entries; should return nil
	schema := sg.handleEnumType("nonexistent.Type")
	AssertEqual(t, nil, schema)
}

// TestHandleEnumType_Positive tests that a string-based enum with constants is converted to a Schema.
func TestHandleEnumType_Positive(t *testing.T) {
	sg := NewTestSchemaGenerator()
	// MyEnum defined in schema_enums_example.go
	schema := sg.handleEnumType("openapi.MyEnum")
	if schema == nil {
		t.Fatal("expected non-nil schema for MyEnum")
	}
	AssertEqual(t, "string", schema.Type)
	AssertDeepEqual(t, []interface{}{"A", "B"}, schema.Enum)
	AssertEqual(t, "Enum type openapi.MyEnum", schema.Description)
}
