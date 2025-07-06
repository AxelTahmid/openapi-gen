package openapi

// ...existing code...

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// ResetGlobals resets any global state for testing.
func ResetGlobals() {
	resetTypeIndexForTesting()
	ensureTypeIndex()
}

// NewTestSchemaGenerator resets globals and returns a SchemaGenerator.
func NewTestSchemaGenerator() *SchemaGenerator {
	ResetGlobals()
	return NewSchemaGenerator()
}

// NewTestGenerator resets globals and returns a Generator.
func NewTestGenerator() *Generator {
	ResetGlobals()
	return NewGenerator()
}

// AssertEqual fails the test if expected != actual.
func AssertEqual[T comparable](t *testing.T, expected, actual T) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected %v, got %v", expected, actual)
	}
}

// AssertDeepEqual fails the test if expected and actual are not deeply equal.
func AssertDeepEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		t.Fatalf("mismatch:\nexpected %#v\nactual   %#v", expected, actual)
	}
}

// AssertNoError fails the test if err is non-nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertJSONEqual fails if two JSON byte slices are not equivalent.
func AssertJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	var a, b interface{}
	if err := json.Unmarshal(want, &a); err != nil {
		t.Fatalf("invalid want JSON: %v", err)
	}
	if err := json.Unmarshal(got, &b); err != nil {
		t.Fatalf("invalid got JSON: %v", err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("JSON mismatch:\nwant %s\ngot  %s", want, got)
	}
}

// Request creates an HTTP request against handler and returns a recorder.
func Request(handler http.Handler, method, path string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, body)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}
