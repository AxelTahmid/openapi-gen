package openapi

import (
	"testing"
)

func TestExtractJSONTag(t *testing.T) {
	tests := []struct {
		tag  string
		want string
	}{
		{"json:\"foo,omitempty\" xml:\"bar\"", "foo"},
		{"json:\"id\"", "id"},
		{`xml:"bar" json:"baz"`, "baz"},
		{"", ""},
	}
	for _, tc := range tests {
		t.Run(tc.tag, func(t *testing.T) {
			got := extractJSONTag(tc.tag)
			AssertEqual(t, tc.want, got)
		})
	}
}

func TestExtractTag(t *testing.T) {
	tests := []struct {
		tag  string
		key  string
		want string
	}{
		{"validate:\"required|min=2\" json:\"f\"", "validate", "required|min=2"},
		{"openapi:\"example=foo\"", "openapi", "example=foo"},
		{"json:\"a\" xml:\"b\"", "binding", ""},
	}
	for _, tc := range tests {
		t.Run(tc.key, func(t *testing.T) {
			got := extractTag(tc.tag, tc.key)
			AssertEqual(t, tc.want, got)
		})
	}
}

func TestApplyEnhancedTags_OpenAPI(t *testing.T) {
	sg := NewTestSchemaGenerator()
	s := &Schema{}

	tag := `openapi:"format=uuid,pattern=^a.*$,enum=a|b|c,default=foo,title=bar,deprecated=true,readOnly=true,writeOnly=true,minimum=1.23,maximum=4.56,minLength=2,maxLength=5,minItems=1,maxItems=3,uniqueItems=true,example=xyz"`
	sg.applyEnhancedTags(s, tag)

	AssertEqual(t, "uuid", s.Format)
	AssertEqual(t, "^a.*$", s.Pattern)
	AssertDeepEqual(t, []interface{}{"a", "b", "c"}, s.Enum)
	AssertEqual(t, "foo", s.Default)
	AssertEqual(t, "bar", s.Title)
	if s.Deprecated == nil || *s.Deprecated != true {
		t.Fatalf("expected Deprecated=true, got %v", s.Deprecated)
	}
	if s.ReadOnly == nil || *s.ReadOnly != true {
		t.Fatalf("expected ReadOnly=true, got %v", s.ReadOnly)
	}
	if s.WriteOnly == nil || *s.WriteOnly != true {
		t.Fatalf("expected WriteOnly=true, got %v", s.WriteOnly)
	}
	if s.Minimum == nil || *s.Minimum != 1.23 {
		t.Fatalf("expected Minimum=1.23, got %v", s.Minimum)
	}
	if s.Maximum == nil || *s.Maximum != 4.56 {
		t.Fatalf("expected Maximum=4.56, got %v", s.Maximum)
	}
	if s.MinLength == nil || *s.MinLength != 2 {
		t.Fatalf("expected MinLength=2, got %v", s.MinLength)
	}
	if s.MaxLength == nil || *s.MaxLength != 5 {
		t.Fatalf("expected MaxLength=5, got %v", s.MaxLength)
	}
	if s.MinItems == nil || *s.MinItems != 1 {
		t.Fatalf("expected MinItems=1, got %v", s.MinItems)
	}
	if s.MaxItems == nil || *s.MaxItems != 3 {
		t.Fatalf("expected MaxItems=3, got %v", s.MaxItems)
	}
	if s.UniqueItems == nil || *s.UniqueItems != true {
		t.Fatalf("expected UniqueItems=true, got %v", s.UniqueItems)
	}
	AssertEqual(t, "xyz", s.Example)
}

func TestApplyEnhancedTags_ValidateBinding(t *testing.T) {
	sg := NewTestSchemaGenerator()
	s := &Schema{}

	tag := `validate:"email" binding:"uuid"`
	sg.applyEnhancedTags(s, tag)
	// binding should override validate
	AssertEqual(t, "uuid", s.Format)
}
