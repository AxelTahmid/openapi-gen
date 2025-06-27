package openapi

import (
	"testing"
)

// --- Test Handlers for annotation parsing ---
// Handler with all annotation types
// @Summary Test summary
// @Description Test description
// @Tags foo,bar
// @Param id path int true "ID param"
// @Param q query string false "Query param"
// @Success 200 {object} TestResponse "Success desc"
// @Failure 400 {object} ProblemDetails "Bad request"
func HandlerWithAnnotations() {}

func TestParseAnnotations_AllAnnotations(t *testing.T) {
	annotation := ParseAnnotations("annotations_test.go", "HandlerWithAnnotations")
	if annotation == nil {
		t.Fatal("ParseAnnotations returned nil")
	}
	if annotation.Summary != "Test summary" {
		t.Errorf("expected summary, got %q", annotation.Summary)
	}
	if annotation.Description != "Test description" {
		t.Errorf("expected description, got %q", annotation.Description)
	}
	if len(annotation.Tags) != 2 || annotation.Tags[0] != "foo" || annotation.Tags[1] != "bar" {
		t.Errorf("expected tags [foo bar], got %+v", annotation.Tags)
	}
	if len(annotation.Parameters) != 2 {
		t.Errorf("expected 2 parameters, got %+v", annotation.Parameters)
	}
	if annotation.Success == nil || annotation.Success.DataType != "TestResponse" {
		t.Errorf("expected success DataType 'TestResponse', got %+v", annotation.Success)
	}
	if len(annotation.Failures) != 1 || annotation.Failures[0].StatusCode != 400 {
		t.Errorf("expected failure 400, got %+v", annotation.Failures)
	}
}

func TestParseAnnotations_Empty(t *testing.T) {
	annotation := ParseAnnotations("annotations_test.go", "NonExistentHandler")
	if annotation != nil {
		t.Error("expected nil for non-existent handler")
	}
}

func Test_parseParamAnnotation(t *testing.T) {
	line := "@Param foo query int true \"desc\""
	param := parseParamAnnotation(line)
	if param == nil || param.Name != "foo" || param.In != "query" || param.Type != "int" || !param.Required ||
		param.Description != "desc" {
		t.Errorf("unexpected param: %+v", param)
	}
}

func Test_parseSuccessAnnotation(t *testing.T) {
	line := "@Success 201 {object} Foo \"desc\""
	succ := parseSuccessAnnotation(line)
	if succ == nil || succ.StatusCode != 201 || succ.DataType != "Foo" || succ.Description != "desc" {
		t.Errorf("unexpected success: %+v", succ)
	}
}

func Test_parseFailureAnnotation(t *testing.T) {
	line := "@Failure 404 {object} Bar \"not found\""
	fail := parseFailureAnnotation(line)
	if fail == nil || fail.StatusCode != 404 || fail.Description != "not found" {
		t.Errorf("unexpected failure: %+v", fail)
	}
}
