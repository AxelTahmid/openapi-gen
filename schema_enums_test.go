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
