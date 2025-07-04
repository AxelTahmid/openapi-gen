package openapi

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// ValidateOpenAPI31Compliance checks if the generated spec meets OpenAPI 3.1 requirements
func ValidateOpenAPI31Compliance(spec Spec) []string {
	var issues []string
	
	// Check OpenAPI version
	if spec.OpenAPI != "3.1.0" {
		issues = append(issues, fmt.Sprintf("OpenAPI version should be '3.1.0', got '%s'", spec.OpenAPI))
	}
	
	// Check JSON Schema dialect (OpenAPI 3.1 specific)
	if spec.JSONSchemaDialect == "" {
		issues = append(issues, "JSONSchemaDialect should be set for OpenAPI 3.1")
	}
	
	// Check that schemas use JSON Schema Draft 2020-12 features
	if spec.Components != nil && spec.Components.Schemas != nil {
		for name, schema := range spec.Components.Schemas {
			schemaIssues := validateSchemaCompliance(name, schema)
			issues = append(issues, schemaIssues...)
		}
	}
	
	// Check webhook support (OpenAPI 3.1 feature)
	if spec.Webhooks != nil && len(spec.Webhooks) > 0 {
		for name, pathItem := range spec.Webhooks {
			if pathItem == nil {
				issues = append(issues, fmt.Sprintf("Webhook '%s' has nil PathItem", name))
			}
		}
	}
	
	return issues
}

// validateSchemaCompliance checks if a schema uses proper OpenAPI 3.1 features
func validateSchemaCompliance(name string, schema Schema) []string {
	var issues []string
	
	// Check for proper use of nullable fields (OpenAPI 3.1 uses oneOf with null)
	if hasNullablePattern(schema) {
		// This is good - using oneOf with null for nullable fields
	}
	
	// Check for deprecated nullable field (should use oneOf instead)
	v := reflect.ValueOf(schema)
	t := reflect.TypeOf(schema)
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.Name == "Nullable" {
			issues = append(issues, fmt.Sprintf("Schema '%s' uses deprecated 'nullable' field, use oneOf with null type instead", name))
		}
	}
	
	// Check for proper validation constraints
	if schema.Type == "string" {
		if schema.MinLength == nil && schema.MaxLength == nil && schema.Pattern == "" && schema.Format == "" {
			// Could suggest adding validation constraints
		}
	}
	
	if schema.Type == "array" {
		if schema.Items == nil {
			issues = append(issues, fmt.Sprintf("Array schema '%s' should have 'items' defined", name))
		}
	}
	
	return issues
}

// hasNullablePattern checks if schema correctly uses oneOf with null for nullable fields
func hasNullablePattern(schema Schema) bool {
	if schema.OneOf == nil || len(schema.OneOf) != 2 {
		return false
	}
	
	// Check if one of the oneOf options is null type
	for _, option := range schema.OneOf {
		if option.Type == "null" {
			return true
		}
	}
	
	return false
}

// GenerateOpenAPI31Summary provides a summary of OpenAPI 3.1 features used
func GenerateOpenAPI31Summary(spec Spec) map[string]interface{} {
	summary := make(map[string]interface{})
	
	// Basic info
	summary["openapi_version"] = spec.OpenAPI
	summary["json_schema_dialect"] = spec.JSONSchemaDialect
	summary["has_webhooks"] = spec.Webhooks != nil && len(spec.Webhooks) > 0
	
	// Count features
	featureCounts := make(map[string]int)
	
	if spec.Components != nil {
		if spec.Components.Schemas != nil {
			featureCounts["schemas"] = len(spec.Components.Schemas)
			
			// Count advanced schema features
			var oneOfCount, anyOfCount, allOfCount, nullableCount int
			var withFormat, withPattern, withEnum int
			
			for _, schema := range spec.Components.Schemas {
				if schema.OneOf != nil && len(schema.OneOf) > 0 {
					oneOfCount++
				}
				if schema.AnyOf != nil && len(schema.AnyOf) > 0 {
					anyOfCount++
				}
				if schema.AllOf != nil && len(schema.AllOf) > 0 {
					allOfCount++
				}
				if hasNullablePattern(schema) {
					nullableCount++
				}
				if schema.Format != "" {
					withFormat++
				}
				if schema.Pattern != "" {
					withPattern++
				}
				if schema.Enum != nil && len(schema.Enum) > 0 {
					withEnum++
				}
			}
			
			featureCounts["schemas_with_oneOf"] = oneOfCount
			featureCounts["schemas_with_anyOf"] = anyOfCount
			featureCounts["schemas_with_allOf"] = allOfCount
			featureCounts["schemas_with_nullable_pattern"] = nullableCount
			featureCounts["schemas_with_format"] = withFormat
			featureCounts["schemas_with_pattern"] = withPattern
			featureCounts["schemas_with_enum"] = withEnum
		}
		
		if spec.Components.Examples != nil {
			featureCounts["examples"] = len(spec.Components.Examples)
		}
		if spec.Components.Links != nil {
			featureCounts["links"] = len(spec.Components.Links)
		}
		if spec.Components.Callbacks != nil {
			featureCounts["callbacks"] = len(spec.Components.Callbacks)
		}
		if spec.Components.Headers != nil {
			featureCounts["headers"] = len(spec.Components.Headers)
		}
	}
	
	if spec.Webhooks != nil {
		featureCounts["webhooks"] = len(spec.Webhooks)
	}
	
	summary["feature_counts"] = featureCounts
	
	// Compliance check
	issues := ValidateOpenAPI31Compliance(spec)
	summary["compliance_issues"] = issues
	summary["is_compliant"] = len(issues) == 0
	
	return summary
}

// PrettyPrintSpec outputs a formatted JSON representation of the spec
func PrettyPrintSpec(spec Spec) (string, error) {
	data, err := json.MarshalIndent(spec, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal spec: %w", err)
	}
	return string(data), nil
}

// GetOpenAPI31FeatureUsage returns a report of which OpenAPI 3.1 features are being used
func GetOpenAPI31FeatureUsage(spec Spec) map[string]bool {
	features := map[string]bool{
		"openapi_3.1_version":     spec.OpenAPI == "3.1.0",
		"json_schema_dialect":     spec.JSONSchemaDialect != "",
		"webhooks":                spec.Webhooks != nil && len(spec.Webhooks) > 0,
		"oneOf_composition":       false,
		"anyOf_composition":       false,
		"allOf_composition":       false,
		"schema_examples":         false,
		"response_links":          false,
		"response_headers":        false,
		"request_encoding":        false,
		"nullable_with_oneOf":     false,
		"string_patterns":         false,
		"numeric_ranges":          false,
		"array_constraints":       false,
		"enum_values":             false,
		"deprecated_schemas":      false,
		"readonly_writeonly":      false,
	}
	
	if spec.Components != nil {
		// Check schemas for advanced features
		if spec.Components.Schemas != nil {
			for _, schema := range spec.Components.Schemas {
				if schema.OneOf != nil && len(schema.OneOf) > 0 {
					features["oneOf_composition"] = true
					if hasNullablePattern(schema) {
						features["nullable_with_oneOf"] = true
					}
				}
				if schema.AnyOf != nil && len(schema.AnyOf) > 0 {
					features["anyOf_composition"] = true
				}
				if schema.AllOf != nil && len(schema.AllOf) > 0 {
					features["allOf_composition"] = true
				}
				if schema.Examples != nil && len(schema.Examples) > 0 {
					features["schema_examples"] = true
				}
				if schema.Pattern != "" {
					features["string_patterns"] = true
				}
				if schema.Minimum != nil || schema.Maximum != nil {
					features["numeric_ranges"] = true
				}
				if schema.MinItems != nil || schema.MaxItems != nil || schema.UniqueItems != nil {
					features["array_constraints"] = true
				}
				if schema.Enum != nil && len(schema.Enum) > 0 {
					features["enum_values"] = true
				}
				if schema.Deprecated != nil && *schema.Deprecated {
					features["deprecated_schemas"] = true
				}
				if (schema.ReadOnly != nil && *schema.ReadOnly) || (schema.WriteOnly != nil && *schema.WriteOnly) {
					features["readonly_writeonly"] = true
				}
			}
		}
		
		// Check responses for links and headers
		if spec.Components.Responses != nil {
			for _, response := range spec.Components.Responses {
				if response.Links != nil && len(response.Links) > 0 {
					features["response_links"] = true
				}
				if response.Headers != nil && len(response.Headers) > 0 {
					features["response_headers"] = true
				}
			}
		}
	}
	
	// Check paths for response features
	for _, pathItem := range spec.Paths {
		for _, operation := range pathItem {
			op := operation
			for _, response := range op.Responses {
				if response.Links != nil && len(response.Links) > 0 {
					features["response_links"] = true
				}
				if response.Headers != nil && len(response.Headers) > 0 {
					features["response_headers"] = true
				}
				for _, content := range response.Content {
					if content.Encoding != nil && len(content.Encoding) > 0 {
						features["request_encoding"] = true
					}
				}
			}
		}
	}
	
	return features
}
