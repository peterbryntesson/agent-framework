// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// FunctionTool wraps a Go function to make it callable by AI models.
// This implementation aligns with Python FunctionTool and .NET AIFunctionFactory.
//
// FunctionTool uses reflection to:
//   - Validate the function signature at creation time
//   - Generate a JSON Schema from the input struct type
//   - Parse JSON arguments and invoke the function
//
// The wrapped function must have one of these signatures:
//   - func(ctx context.Context, args T) (R, error)
//   - func(ctx context.Context, args T) error
//   - func(ctx context.Context, args T) R
//   - func(args T) (R, error)
//   - func(args T) error
//   - func(args T) R
//   - func(ctx context.Context) (R, error) - for no-arg functions
//   - func(ctx context.Context) error
//   - func() (R, error)
//   - func() error
//
// Where T is a struct with json tags defining parameter names and descriptions,
// and R is any type that can be converted to a Result.
type FunctionTool struct {
	name        string
	description string
	fn          reflect.Value
	fnType      reflect.Type
	inputType   reflect.Type
	schema      json.RawMessage
	hasContext  bool
	hasArgs     bool
	hasError    bool

	// Configuration options
	ApprovalMode   ApprovalMode
	MaxInvocations int
	Properties     AdditionalProperties

	// Runtime state
	invocationCount int
}

// FunctionToolConfig contains configuration for creating a FunctionTool.
type FunctionToolConfig struct {
	Name           string
	Description    string
	ApprovalMode   ApprovalMode
	MaxInvocations int
	Properties     AdditionalProperties
}

// ErrInvalidSignature is returned when a function has an unsupported signature.
var ErrInvalidSignature = errors.New("invalid function signature")

// ErrMaxInvocationsReached is returned when a tool has reached its invocation limit.
var ErrMaxInvocationsReached = errors.New("maximum invocations reached")

// ErrNilFunction is returned when attempting to create a FunctionTool with a nil function.
var ErrNilFunction = errors.New("function cannot be nil")

// NewFunctionTool creates a FunctionTool from a Go function.
// The function signature is validated at creation time, and a JSON Schema
// is generated from the input struct type if present.
//
// See FunctionTool documentation for supported function signatures.
//
// Example:
//
//	type WeatherArgs struct {
//	    Location string `json:"location" description:"The city name" required:"true"`
//	    Unit     string `json:"unit" description:"Temperature unit" enum:"celsius,fahrenheit"`
//	}
//
//	func getWeather(ctx context.Context, args WeatherArgs) (string, error) {
//	    return fmt.Sprintf("Weather in %s: 22°%s", args.Location, args.Unit), nil
//	}
//
//	tool, err := NewFunctionTool("get_weather", "Get the weather for a location", getWeather)
func NewFunctionTool(name, description string, fn interface{}) (*FunctionTool, error) {
	if fn == nil {
		return nil, ErrNilFunction
	}

	fnValue := reflect.ValueOf(fn)
	fnType := fnValue.Type()

	if fnType.Kind() != reflect.Func {
		return nil, fmt.Errorf("%w: expected function, got %s", ErrInvalidSignature, fnType.Kind())
	}

	tool := &FunctionTool{
		name:        name,
		description: description,
		fn:          fnValue,
		fnType:      fnType,
	}

	// Analyze the function signature
	if err := tool.analyzeSignature(); err != nil {
		return nil, err
	}

	// Generate schema from input type if present
	if tool.inputType != nil {
		schema, err := GenerateSchema(tool.inputType)
		if err != nil {
			return nil, fmt.Errorf("failed to generate schema: %w", err)
		}
		tool.schema = schema
	} else {
		// Empty object schema for no-arg functions
		tool.schema = json.RawMessage(`{"type":"object","properties":{}}`)
	}

	return tool, nil
}

// analyzeSignature validates the function signature and extracts type information.
func (t *FunctionTool) analyzeSignature() error {
	numIn := t.fnType.NumIn()
	numOut := t.fnType.NumOut()

	// Validate output count (0-2 outputs allowed)
	if numOut > 2 {
		return fmt.Errorf("%w: too many return values (max 2)", ErrInvalidSignature)
	}

	// Check if last return value is error
	if numOut > 0 {
		lastOut := t.fnType.Out(numOut - 1)
		t.hasError = lastOut.Implements(reflect.TypeOf((*error)(nil)).Elem())
	}

	// Validate return types
	if numOut == 2 && !t.hasError {
		return fmt.Errorf("%w: second return value must be error", ErrInvalidSignature)
	}

	// Analyze input parameters
	paramIdx := 0

	// Check for context.Context as first parameter
	if numIn > 0 {
		firstIn := t.fnType.In(0)
		if firstIn.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
			t.hasContext = true
			paramIdx++
		}
	}

	// Check for args struct as next parameter
	if paramIdx < numIn {
		argsType := t.fnType.In(paramIdx)

		// Dereference pointer types
		if argsType.Kind() == reflect.Ptr {
			argsType = argsType.Elem()
		}

		// Args must be a struct
		if argsType.Kind() != reflect.Struct {
			return fmt.Errorf("%w: args parameter must be a struct, got %s", ErrInvalidSignature, argsType.Kind())
		}

		t.inputType = t.fnType.In(paramIdx) // Keep original (possibly pointer) type
		t.hasArgs = true
		paramIdx++
	}

	// No additional parameters allowed
	if paramIdx < numIn {
		return fmt.Errorf("%w: unexpected parameters after args", ErrInvalidSignature)
	}

	return nil
}

// Name returns the unique identifier for this tool.
func (t *FunctionTool) Name() string {
	return t.name
}

// Description returns a human-readable description of what this tool does.
func (t *FunctionTool) Description() string {
	return t.description
}

// Parameters returns the JSON Schema describing the tool's input parameters.
func (t *FunctionTool) Parameters() json.RawMessage {
	return t.schema
}

// Invoke executes the tool with the given arguments.
// Arguments are provided as raw JSON matching the Parameters schema.
func (t *FunctionTool) Invoke(ctx context.Context, arguments json.RawMessage) (Result, error) {
	// Check invocation limit
	if t.MaxInvocations > 0 && t.invocationCount >= t.MaxInvocations {
		return Result{}, ErrMaxInvocationsReached
	}
	t.invocationCount++

	// Build input arguments for the function call
	inputs, err := t.buildInputs(ctx, arguments)
	if err != nil {
		return NewErrorResult(fmt.Sprintf("failed to parse arguments: %v", err)), nil
	}

	// Call the function
	outputs := t.fn.Call(inputs)

	// Process outputs
	return t.processOutputs(outputs)
}

// buildInputs constructs the reflect.Value slice for calling the function.
func (t *FunctionTool) buildInputs(ctx context.Context, arguments json.RawMessage) ([]reflect.Value, error) {
	var inputs []reflect.Value

	// Add context if required
	if t.hasContext {
		inputs = append(inputs, reflect.ValueOf(ctx))
	}

	// Add args if required
	if t.hasArgs {
		argsValue, err := t.parseArguments(arguments)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, argsValue)
	}

	return inputs, nil
}

// parseArguments parses JSON arguments into the function's args type.
func (t *FunctionTool) parseArguments(arguments json.RawMessage) (reflect.Value, error) {
	// Handle nil or empty arguments
	if len(arguments) == 0 || string(arguments) == "null" || string(arguments) == "{}" {
		// Create zero value of the input type
		if t.inputType.Kind() == reflect.Ptr {
			return reflect.New(t.inputType.Elem()), nil
		}
		return reflect.New(t.inputType).Elem(), nil
	}

	// Create a new instance of the input type
	var argsPtr reflect.Value
	if t.inputType.Kind() == reflect.Ptr {
		argsPtr = reflect.New(t.inputType.Elem())
	} else {
		argsPtr = reflect.New(t.inputType)
	}

	// Unmarshal JSON into the args
	if err := json.Unmarshal(arguments, argsPtr.Interface()); err != nil {
		return reflect.Value{}, fmt.Errorf("JSON unmarshal failed: %w", err)
	}

	// Return the value with correct type (pointer or value)
	if t.inputType.Kind() == reflect.Ptr {
		return argsPtr, nil
	}
	return argsPtr.Elem(), nil
}

// processOutputs converts the function's return values to a Result.
func (t *FunctionTool) processOutputs(outputs []reflect.Value) (Result, error) {
	// Handle functions with no return values
	if len(outputs) == 0 {
		return NewResult(""), nil
	}

	// Handle error return value
	if t.hasError {
		errIdx := len(outputs) - 1
		errVal := outputs[errIdx]
		if !errVal.IsNil() {
			err := errVal.Interface().(error)
			return NewErrorResult(err.Error()), nil
		}
		outputs = outputs[:errIdx] // Remove error from outputs for result processing
	}

	// Handle result value
	if len(outputs) == 0 {
		return NewResult(""), nil
	}

	resultVal := outputs[0]
	if !resultVal.IsValid() || (resultVal.Kind() == reflect.Ptr && resultVal.IsNil()) {
		return NewResult(""), nil
	}

	// Convert to Result
	return NewResultFromValue(resultVal.Interface())
}

// InvocationCount returns the number of times this tool has been invoked.
func (t *FunctionTool) InvocationCount() int {
	return t.invocationCount
}

// ResetInvocationCount resets the invocation counter to zero.
func (t *FunctionTool) ResetInvocationCount() {
	t.invocationCount = 0
}

// FuncOption configures a FunctionTool created via Func().
type FuncOption func(*FunctionTool)

// WithName sets a custom name for the tool (default: function name).
func WithName(name string) FuncOption {
	return func(t *FunctionTool) {
		t.name = name
	}
}

// WithDescription sets the description for the tool.
func WithDescription(description string) FuncOption {
	return func(t *FunctionTool) {
		t.description = description
	}
}

// WithApprovalMode sets whether user approval is required.
func WithApprovalMode(mode ApprovalMode) FuncOption {
	return func(t *FunctionTool) {
		t.ApprovalMode = mode
	}
}

// WithMaxInvocations limits how many times this tool can be called.
func WithMaxInvocations(max int) FuncOption {
	return func(t *FunctionTool) {
		t.MaxInvocations = max
	}
}

// WithProperties sets additional properties on the tool.
func WithProperties(props AdditionalProperties) FuncOption {
	return func(t *FunctionTool) {
		t.Properties = props
	}
}

// Func creates a FunctionTool from a function with options.
// This provides a convenient decorator-style syntax for creating tools.
//
// If no name is provided via WithName, the function's runtime name is used.
// If no description is provided, an empty description is used.
//
// Example:
//
//	weatherTool := tool.Func(getWeather,
//	    tool.WithDescription("Get the weather for a location"),
//	    tool.WithApprovalMode(tool.ApprovalNever),
//	)
func Func(fn interface{}, opts ...FuncOption) (*FunctionTool, error) {
	// Extract function name from runtime
	name := extractFunctionName(fn)

	// Create the tool with defaults
	tool, err := NewFunctionTool(name, "", fn)
	if err != nil {
		return nil, err
	}

	// Apply options
	for _, opt := range opts {
		opt(tool)
	}

	return tool, nil
}

// MustFunc is like Func but panics on error.
// Use this when you're certain the function signature is valid.
func MustFunc(fn interface{}, opts ...FuncOption) *FunctionTool {
	tool, err := Func(fn, opts...)
	if err != nil {
		panic(fmt.Sprintf("tool.MustFunc: %v", err))
	}
	return tool
}

// extractFunctionName attempts to extract the function name using runtime reflection.
func extractFunctionName(fn interface{}) string {
	fnValue := reflect.ValueOf(fn)
	if fnValue.Kind() != reflect.Func {
		return "unknown"
	}

	// Get the function pointer
	ptr := fnValue.Pointer()
	fnInfo := runtime.FuncForPC(ptr)
	if fnInfo == nil {
		return "unknown"
	}

	// Extract just the function name from the full path
	fullName := fnInfo.Name()
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		name := parts[len(parts)-1]
		// Remove any method receiver suffix (e.g., "-fm" for methods)
		name = strings.TrimSuffix(name, "-fm")
		return name
	}

	return fullName
}

// Compile-time check that FunctionTool implements Tool interface.
var _ Tool = (*FunctionTool)(nil)
