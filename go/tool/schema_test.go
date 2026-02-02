// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Test struct types for schema generation tests.
type basicStruct struct {
	StringField string `json:"string_field" description:"A string field" required:"true"`
	IntField    int    `json:"int_field" description:"An integer field"`
	BoolField   bool   `json:"bool_field"`
}

type numericStruct struct {
	Int8Field    int8    `json:"int8_field"`
	Int16Field   int16   `json:"int16_field"`
	Int32Field   int32   `json:"int32_field"`
	Int64Field   int64   `json:"int64_field"`
	UintField    uint    `json:"uint_field"`
	Uint8Field   uint8   `json:"uint8_field"`
	Uint16Field  uint16  `json:"uint16_field"`
	Uint32Field  uint32  `json:"uint32_field"`
	Uint64Field  uint64  `json:"uint64_field"`
	Float32Field float32 `json:"float32_field"`
	Float64Field float64 `json:"float64_field"`
}

type enumStruct struct {
	Status string `json:"status" enum:"pending,active,completed"`
	Unit   string `json:"unit" enum:"celsius,fahrenheit,kelvin"`
}

type defaultStruct struct {
	StringDefault string  `json:"string_default" default:"hello"`
	IntDefault    int     `json:"int_default" default:"42"`
	BoolDefault   bool    `json:"bool_default" default:"true"`
	FloatDefault  float64 `json:"float_default" default:"3.14"`
}

type nestedStruct struct {
	Name   string       `json:"name" required:"true"`
	Inner  innerStruct  `json:"inner"`
	InnerP *innerStruct `json:"inner_p,omitempty"`
}

type innerStruct struct {
	Value string `json:"value" required:"true"`
	Count int    `json:"count"`
}

type sliceStruct struct {
	Tags    []string      `json:"tags" description:"List of tags"`
	Numbers []int         `json:"numbers"`
	Nested  []innerStruct `json:"nested"`
}

type mapStruct struct {
	Labels   map[string]string `json:"labels"`
	Counts   map[string]int    `json:"counts"`
	Metadata map[string]any    `json:"metadata"`
}

type optionalFieldsStruct struct {
	Required    string  `json:"required" required:"true"`
	Optional    *string `json:"optional"`
	Omitempty   string  `json:"omitempty,omitempty"`
	ExplicitOpt string  `json:"explicit_opt" required:"false"`
}

type unexportedFieldsStruct struct {
	Public  string `json:"public"`
	private string
}

type jsonTagsStruct struct {
	Normal   string `json:"normal_name"`
	Excluded string `json:"-"`
	NoTag    string
}

func TestGenerateSchema_BasicTypes(t *testing.T) {
	t.Run("generates schema for basic struct", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(basicStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		if schema["type"] != "object" {
			t.Errorf("expected type 'object', got %v", schema["type"])
		}

		props, ok := schema["properties"].(map[string]interface{})
		if !ok {
			t.Fatal("expected properties to be a map")
		}

		// Check string field
		stringField, ok := props["string_field"].(map[string]interface{})
		if !ok {
			t.Fatal("expected string_field property")
		}
		if stringField["type"] != "string" {
			t.Errorf("expected string_field type 'string', got %v", stringField["type"])
		}
		if stringField["description"] != "A string field" {
			t.Errorf("expected description 'A string field', got %v", stringField["description"])
		}

		// Check int field
		intField, ok := props["int_field"].(map[string]interface{})
		if !ok {
			t.Fatal("expected int_field property")
		}
		if intField["type"] != "integer" {
			t.Errorf("expected int_field type 'integer', got %v", intField["type"])
		}

		// Check bool field
		boolField, ok := props["bool_field"].(map[string]interface{})
		if !ok {
			t.Fatal("expected bool_field property")
		}
		if boolField["type"] != "boolean" {
			t.Errorf("expected bool_field type 'boolean', got %v", boolField["type"])
		}
	})

	t.Run("generates schema for numeric types", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(numericStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// All integer types should be "integer"
		intTypes := []string{"int8_field", "int16_field", "int32_field", "int64_field",
			"uint_field", "uint8_field", "uint16_field", "uint32_field", "uint64_field"}
		for _, name := range intTypes {
			field := props[name].(map[string]interface{})
			if field["type"] != "integer" {
				t.Errorf("expected %s type 'integer', got %v", name, field["type"])
			}
		}

		// Float types should be "number"
		floatTypes := []string{"float32_field", "float64_field"}
		for _, name := range floatTypes {
			field := props[name].(map[string]interface{})
			if field["type"] != "number" {
				t.Errorf("expected %s type 'number', got %v", name, field["type"])
			}
		}
	})
}

func TestGenerateSchema_EnumSupport(t *testing.T) {
	t.Run("parses enum tag", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(enumStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// Check status enum
		statusField := props["status"].(map[string]interface{})
		statusEnum, ok := statusField["enum"].([]interface{})
		if !ok {
			t.Fatal("expected status.enum to be a slice")
		}
		if len(statusEnum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(statusEnum))
		}
		expectedStatus := []string{"pending", "active", "completed"}
		for i, expected := range expectedStatus {
			if statusEnum[i] != expected {
				t.Errorf("expected enum[%d] = %q, got %v", i, expected, statusEnum[i])
			}
		}

		// Check unit enum
		unitField := props["unit"].(map[string]interface{})
		unitEnum := unitField["enum"].([]interface{})
		if len(unitEnum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(unitEnum))
		}
	})
}

func TestGenerateSchema_DefaultValues(t *testing.T) {
	t.Run("parses default tag with correct types", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(defaultStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// String default
		stringField := props["string_default"].(map[string]interface{})
		if stringField["default"] != "hello" {
			t.Errorf("expected string default 'hello', got %v", stringField["default"])
		}

		// Integer default (parsed as float64 in JSON)
		intField := props["int_default"].(map[string]interface{})
		if intField["default"] != float64(42) {
			t.Errorf("expected int default 42, got %v", intField["default"])
		}

		// Boolean default
		boolField := props["bool_default"].(map[string]interface{})
		if boolField["default"] != true {
			t.Errorf("expected bool default true, got %v", boolField["default"])
		}

		// Float default
		floatField := props["float_default"].(map[string]interface{})
		if floatField["default"] != 3.14 {
			t.Errorf("expected float default 3.14, got %v", floatField["default"])
		}
	})
}

func TestGenerateSchema_NestedStructs(t *testing.T) {
	t.Run("generates nested object schema", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(nestedStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// Check nested inner struct
		innerField := props["inner"].(map[string]interface{})
		if innerField["type"] != "object" {
			t.Errorf("expected inner type 'object', got %v", innerField["type"])
		}

		innerProps := innerField["properties"].(map[string]interface{})
		valueField := innerProps["value"].(map[string]interface{})
		if valueField["type"] != "string" {
			t.Errorf("expected value type 'string', got %v", valueField["type"])
		}

		// Check required in nested
		innerRequired := innerField["required"].([]interface{})
		foundValue := false
		for _, r := range innerRequired {
			if r == "value" {
				foundValue = true
				break
			}
		}
		if !foundValue {
			t.Error("expected 'value' to be in inner.required")
		}
	})

	t.Run("handles pointer to nested struct", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(nestedStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// Check pointer to inner struct
		innerPField := props["inner_p"].(map[string]interface{})
		if innerPField["type"] != "object" {
			t.Errorf("expected inner_p type 'object', got %v", innerPField["type"])
		}
	})
}

func TestGenerateSchema_Slices(t *testing.T) {
	t.Run("generates array schema for slices", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(sliceStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// String slice
		tagsField := props["tags"].(map[string]interface{})
		if tagsField["type"] != "array" {
			t.Errorf("expected tags type 'array', got %v", tagsField["type"])
		}
		tagsItems := tagsField["items"].(map[string]interface{})
		if tagsItems["type"] != "string" {
			t.Errorf("expected tags items type 'string', got %v", tagsItems["type"])
		}
		if tagsField["description"] != "List of tags" {
			t.Errorf("expected description 'List of tags', got %v", tagsField["description"])
		}

		// Int slice
		numbersField := props["numbers"].(map[string]interface{})
		if numbersField["type"] != "array" {
			t.Errorf("expected numbers type 'array', got %v", numbersField["type"])
		}
		numbersItems := numbersField["items"].(map[string]interface{})
		if numbersItems["type"] != "integer" {
			t.Errorf("expected numbers items type 'integer', got %v", numbersItems["type"])
		}

		// Nested struct slice
		nestedField := props["nested"].(map[string]interface{})
		if nestedField["type"] != "array" {
			t.Errorf("expected nested type 'array', got %v", nestedField["type"])
		}
		nestedItems := nestedField["items"].(map[string]interface{})
		if nestedItems["type"] != "object" {
			t.Errorf("expected nested items type 'object', got %v", nestedItems["type"])
		}
	})
}

func TestGenerateSchema_Maps(t *testing.T) {
	t.Run("generates object schema for maps", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(mapStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		// String map
		labelsField := props["labels"].(map[string]interface{})
		if labelsField["type"] != "object" {
			t.Errorf("expected labels type 'object', got %v", labelsField["type"])
		}
	})

	t.Run("rejects non-string keys", func(t *testing.T) {
		// Arrange
		type badMap struct {
			IntKeys map[int]string `json:"int_keys"`
		}

		// Act
		_, err := GenerateSchema(reflect.TypeOf(badMap{}))

		// Assert
		if err == nil {
			t.Fatal("expected error for non-string map keys")
		}
	})
}

func TestGenerateSchema_RequiredFields(t *testing.T) {
	t.Run("marks required fields correctly", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(optionalFieldsStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		required := schema["required"].([]interface{})

		// Check required field is in the list
		foundRequired := false
		for _, r := range required {
			if r == "required" {
				foundRequired = true
			}
			// These should NOT be in required
			if r == "optional" || r == "omitempty" || r == "explicit_opt" {
				t.Errorf("field %v should not be required", r)
			}
		}

		if !foundRequired {
			t.Error("expected 'required' to be in required list")
		}
	})

	t.Run("pointer fields are optional", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(optionalFieldsStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		required := schema["required"].([]interface{})

		for _, r := range required {
			if r == "optional" {
				t.Error("pointer field should not be required")
			}
		}
	})

	t.Run("omitempty fields are optional", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(optionalFieldsStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		required := schema["required"].([]interface{})

		for _, r := range required {
			if r == "omitempty" {
				t.Error("omitempty field should not be required")
			}
		}
	})
}

func TestGenerateSchema_FieldVisibility(t *testing.T) {
	t.Run("skips unexported fields", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(unexportedFieldsStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		if _, ok := props["public"]; !ok {
			t.Error("expected 'public' field")
		}
		if _, ok := props["private"]; ok {
			t.Error("unexported field should not be included")
		}
	})

	t.Run("respects json:- tag", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(jsonTagsStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		if _, ok := props["normal_name"]; !ok {
			t.Error("expected 'normal_name' field")
		}
		if _, ok := props["Excluded"]; ok {
			t.Error("json:- field should not be included")
		}
		if _, ok := props["-"]; ok {
			t.Error("json:- field should not be included")
		}
	})

	t.Run("uses field name when no json tag", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf(jsonTagsStruct{}))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		props := schema["properties"].(map[string]interface{})

		if _, ok := props["NoTag"]; !ok {
			t.Error("expected 'NoTag' field (Go field name used when no json tag)")
		}
	})
}

func TestGenerateSchema_PointerTypes(t *testing.T) {
	t.Run("handles top-level pointer type", func(t *testing.T) {
		// Act
		schemaBytes, err := GenerateSchema(reflect.TypeOf((*basicStruct)(nil)))

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var schema map[string]interface{}
		if err := json.Unmarshal(schemaBytes, &schema); err != nil {
			t.Fatalf("failed to unmarshal schema: %v", err)
		}

		if schema["type"] != "object" {
			t.Errorf("expected type 'object', got %v", schema["type"])
		}
	})
}

func TestGenerateSchema_UnsupportedTypes(t *testing.T) {
	t.Run("returns error for channel type", func(t *testing.T) {
		// Arrange
		type badStruct struct {
			Ch chan int `json:"ch"`
		}

		// Act
		_, err := GenerateSchema(reflect.TypeOf(badStruct{}))

		// Assert
		if err == nil {
			t.Fatal("expected error for channel type")
		}
	})

	t.Run("returns error for func type", func(t *testing.T) {
		// Arrange
		type badStruct struct {
			Fn func() `json:"fn"`
		}

		// Act
		_, err := GenerateSchema(reflect.TypeOf(badStruct{}))

		// Assert
		if err == nil {
			t.Fatal("expected error for func type")
		}
	})
}

func TestSchemaFromJSON(t *testing.T) {
	t.Run("parses valid JSON schema", func(t *testing.T) {
		// Arrange
		data := []byte(`{"type":"object","properties":{"name":{"type":"string"}}}`)

		// Act
		schema, err := SchemaFromJSON(data)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if schema.Type != "object" {
			t.Errorf("expected type 'object', got %q", schema.Type)
		}
		if schema.Properties["name"] == nil {
			t.Error("expected 'name' property")
		}
		if schema.Properties["name"].Type != "string" {
			t.Errorf("expected name type 'string', got %q", schema.Properties["name"].Type)
		}
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		// Act
		_, err := SchemaFromJSON([]byte(`{invalid`))

		// Assert
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
	})
}

func TestSchema_ToRawMessage(t *testing.T) {
	t.Run("converts schema to RawMessage", func(t *testing.T) {
		// Arrange
		schema := &Schema{
			Type:        "object",
			Description: "Test schema",
			Properties: map[string]*Schema{
				"name": {Type: "string", Description: "The name"},
			},
			Required: []string{"name"},
		}

		// Act
		raw, err := schema.ToRawMessage()

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if parsed["type"] != "object" {
			t.Errorf("expected type 'object', got %v", parsed["type"])
		}
		if parsed["description"] != "Test schema" {
			t.Errorf("expected description 'Test schema', got %v", parsed["description"])
		}
	})
}

func TestSchema_MarshalJSON(t *testing.T) {
	t.Run("marshals schema correctly", func(t *testing.T) {
		// Arrange
		schema := &Schema{
			Type: "string",
			Enum: []string{"a", "b", "c"},
		}

		// Act
		data, err := json.Marshal(schema)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if parsed["type"] != "string" {
			t.Errorf("expected type 'string', got %v", parsed["type"])
		}

		enum := parsed["enum"].([]interface{})
		if len(enum) != 3 {
			t.Errorf("expected 3 enum values, got %d", len(enum))
		}
	})

	t.Run("omits empty fields", func(t *testing.T) {
		// Arrange
		schema := &Schema{
			Type: "string",
		}

		// Act
		data, err := json.Marshal(schema)

		// Assert
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal(data, &parsed); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if _, ok := parsed["properties"]; ok {
			t.Error("expected properties to be omitted when nil")
		}
		if _, ok := parsed["required"]; ok {
			t.Error("expected required to be omitted when nil")
		}
	})
}
