// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// Schema represents a JSON Schema for tool parameters.
// This is a simplified JSON Schema representation suitable for LLM tool definitions.
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Description          string             `json:"description,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Enum                 []string           `json:"enum,omitempty"`
	Default              interface{}        `json:"default,omitempty"`
	AdditionalProperties *bool              `json:"additionalProperties,omitempty"`
}

// GenerateSchema creates a JSON Schema from a Go struct type.
// The function uses reflection to inspect the type and generates a schema
// that describes the structure for LLM tool calling.
//
// Supported struct tags:
//   - `json:"name"` for property names (standard Go JSON tag)
//   - `description:"text"` for property descriptions
//   - `required:"true"` for required properties
//   - `enum:"a,b,c"` for enumerated values
//   - `default:"value"` for default values
//
// Supported types:
//   - string, bool, int/int8/int16/int32/int64
//   - uint/uint8/uint16/uint32/uint64
//   - float32, float64
//   - slices and arrays
//   - nested structs
//   - pointers (treated as optional unless tagged required)
//   - map[string]T (treated as object with additionalProperties)
//
// Example:
//
//	type WeatherArgs struct {
//	    Location string `json:"location" description:"The city name" required:"true"`
//	    Unit     string `json:"unit" description:"Temperature unit" enum:"celsius,fahrenheit"`
//	}
//
//	schema, err := GenerateSchema(reflect.TypeOf(WeatherArgs{}))
func GenerateSchema(t reflect.Type) (json.RawMessage, error) {
	schema, err := generateSchemaForType(t)
	if err != nil {
		return nil, err
	}

	bytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	return bytes, nil
}

// generateSchemaForType creates a Schema for the given reflect.Type.
func generateSchemaForType(t reflect.Type) (*Schema, error) {
	// Dereference pointer types
	if t.Kind() == reflect.Ptr {
		return generateSchemaForType(t.Elem())
	}

	switch t.Kind() {
	case reflect.Struct:
		return generateStructSchema(t)
	case reflect.Slice, reflect.Array:
		return generateArraySchema(t)
	case reflect.Map:
		return generateMapSchema(t)
	case reflect.String:
		return &Schema{Type: "string"}, nil
	case reflect.Bool:
		return &Schema{Type: "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}, nil
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}, nil
	case reflect.Interface:
		// interface{} or any - no type constraint
		return &Schema{}, nil
	default:
		return nil, fmt.Errorf("unsupported type: %s", t.Kind())
	}
}

// generateStructSchema creates a Schema for a struct type.
func generateStructSchema(t reflect.Type) (*Schema, error) {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
		Required:   []string{},
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		// Get the JSON name for the field
		name := getJSONFieldName(field)
		if name == "-" {
			continue // Field is explicitly excluded from JSON
		}

		// Generate schema for the field type
		fieldSchema, err := generateSchemaForType(field.Type)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		// Apply struct tags to the field schema
		applyFieldTags(field, fieldSchema)

		// Check if field is required
		if isFieldRequired(field) {
			schema.Required = append(schema.Required, name)
		}

		schema.Properties[name] = fieldSchema
	}

	// Clean up empty required slice for cleaner JSON output
	if len(schema.Required) == 0 {
		schema.Required = nil
	}

	return schema, nil
}

// generateArraySchema creates a Schema for slice or array types.
func generateArraySchema(t reflect.Type) (*Schema, error) {
	elemSchema, err := generateSchemaForType(t.Elem())
	if err != nil {
		return nil, fmt.Errorf("array element: %w", err)
	}

	return &Schema{
		Type:  "array",
		Items: elemSchema,
	}, nil
}

// generateMapSchema creates a Schema for map types.
func generateMapSchema(t reflect.Type) (*Schema, error) {
	// Only support string keys
	if t.Key().Kind() != reflect.String {
		return nil, fmt.Errorf("map keys must be strings, got %s", t.Key().Kind())
	}

	// Generate schema for map values
	valueSchema, err := generateSchemaForType(t.Elem())
	if err != nil {
		return nil, fmt.Errorf("map value: %w", err)
	}

	// Maps are represented as objects with additionalProperties
	// For simplicity, we use a boolean to indicate any additional properties are allowed
	// with the type constraint in the items
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	// If the value type has a specific schema, set it
	if valueSchema.Type != "" {
		additionalProps := true
		schema.AdditionalProperties = &additionalProps
		// Note: Full JSON Schema would put valueSchema in additionalProperties
		// but for LLM tool calling, a simpler representation is often preferred
	}

	return schema, nil
}

// getJSONFieldName extracts the JSON field name from struct tags.
func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}

	// Handle json tag format: "name,omitempty"
	parts := strings.Split(tag, ",")
	name := parts[0]

	if name == "" {
		return field.Name
	}

	return name
}

// applyFieldTags applies description, enum, and default tags to a schema.
func applyFieldTags(field reflect.StructField, schema *Schema) {
	// Description tag
	if desc := field.Tag.Get("description"); desc != "" {
		schema.Description = desc
	}

	// Enum tag - comma-separated values
	if enum := field.Tag.Get("enum"); enum != "" {
		schema.Enum = strings.Split(enum, ",")
		// Trim whitespace from each enum value
		for i, v := range schema.Enum {
			schema.Enum[i] = strings.TrimSpace(v)
		}
	}

	// Default tag
	if defaultVal := field.Tag.Get("default"); defaultVal != "" {
		schema.Default = parseDefaultValue(defaultVal, schema.Type)
	}
}

// isFieldRequired determines if a field should be marked as required.
func isFieldRequired(field reflect.StructField) bool {
	// Check explicit required tag
	if req := field.Tag.Get("required"); req != "" {
		return strings.ToLower(req) == "true"
	}

	// Pointer fields are optional by convention
	if field.Type.Kind() == reflect.Ptr {
		return false
	}

	// Check if json tag has "omitempty" - treated as optional
	tag := field.Tag.Get("json")
	if strings.Contains(tag, "omitempty") {
		return false
	}

	// Non-pointer, non-omitempty fields default to required
	return true
}

// parseDefaultValue converts a default tag value to the appropriate type.
func parseDefaultValue(value string, schemaType string) interface{} {
	switch schemaType {
	case "integer":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
		}
	case "number":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	case "boolean":
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	// Default to string for unknown types or parsing failures
	return value
}

// SchemaFromJSON parses a JSON Schema from a JSON string.
func SchemaFromJSON(data []byte) (*Schema, error) {
	var schema Schema
	if err := json.Unmarshal(data, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema: %w", err)
	}
	return &schema, nil
}

// MarshalJSON returns the JSON representation of the schema.
func (s *Schema) MarshalJSON() ([]byte, error) {
	type schemaAlias Schema
	return json.Marshal((*schemaAlias)(s))
}

// ToRawMessage converts the schema to json.RawMessage.
func (s *Schema) ToRawMessage() (json.RawMessage, error) {
	return json.Marshal(s)
}
