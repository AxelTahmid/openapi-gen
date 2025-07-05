package openapi

// newTestSchemaGenerator resets the global TypeIndex for testing, ensures it's initialized,
// and returns a fresh SchemaGenerator.
func newTestSchemaGenerator() *SchemaGenerator {
	resetTypeIndexForTesting()
	ensureTypeIndex()
	return NewSchemaGenerator()
}

// newTestGenerator resets the global TypeIndex for testing and returns a fresh Generator.
func newTestGenerator() *Generator {
	resetTypeIndexForTesting()
	ensureTypeIndex()
	return NewGenerator()
}
