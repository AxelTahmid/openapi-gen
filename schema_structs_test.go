package openapi

import (
	"go/ast"
	"go/parser"
	"testing"
)

func TestIsPointerType(t *testing.T) {
	tests := []struct {
		expr ast.Expr
		want bool
	}{
		{nil, false},
		{&ast.StarExpr{}, true},
	}
	for i, tc := range tests {
		t.Run(string(i), func(t *testing.T) {
			got := isPointerType(tc.expr)
			AssertEqual(t, tc.want, got)
		})
	}
}

func TestHasOmitEmpty(t *testing.T) {
	tests := []struct {
		tag  *ast.BasicLit
		want bool
	}{
		{nil, false},
		{&ast.BasicLit{Value: "`json:\"a,omitempty\"`"}, true},
		{&ast.BasicLit{Value: "`json:\"b\"`"}, false},
	}
	for i, tc := range tests {
		t.Run(string(i), func(t *testing.T) {
			got := hasOmitEmpty(tc.tag)
			AssertEqual(t, tc.want, got)
		})
	}
}

func TestConvertFieldType(t *testing.T) {
	sg := NewTestSchemaGenerator()
	tests := []struct {
		name string
		expr ast.Expr
		want *Schema
	}{
		{"IdentString", &ast.Ident{Name: "string"}, &Schema{Type: "string"}},
		{"PointerBool", &ast.StarExpr{X: &ast.Ident{Name: "bool"}}, &Schema{Type: "boolean"}},
		{"ArrayInt", &ast.ArrayType{Elt: &ast.Ident{Name: "int"}}, &Schema{Type: "array", Items: &Schema{Type: "integer"}}},
		{"MapString", &ast.MapType{Value: &ast.Ident{Name: "string"}}, &Schema{Type: "object", AdditionalProperties: &Schema{Type: "string"}}},
		{"Interface", &ast.InterfaceType{}, &Schema{Type: "object"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sg.convertFieldType(tc.expr)
			AssertDeepEqual(t, tc.want, got)
		})
	}
}

func TestConvertStructToSchema_Simple(t *testing.T) {
	src := `struct {
		A string ` + "`json:\"a\"`" + `
		B *int
		C bool ` + "`json:\"c,omitempty\"`" + `
		d float64
	}`
	expr, err := parser.ParseExpr(src)
	AssertNoError(t, err)
	structType, ok := expr.(*ast.StructType)
	AssertEqual(t, true, ok)

	sg := NewTestSchemaGenerator()
	schema := sg.convertStructToSchema(structType)

	// Basic checks
	AssertEqual(t, "object", schema.Type)
	AssertDeepEqual(t, []string{"a"}, schema.Required)

	// Properties
	AssertDeepEqual(t, &Schema{Type: "string"}, schema.Properties["a"])
	AssertDeepEqual(t, &Schema{Type: "integer"}, schema.Properties["B"])
	AssertDeepEqual(t, &Schema{Type: "boolean"}, schema.Properties["c"])
}
