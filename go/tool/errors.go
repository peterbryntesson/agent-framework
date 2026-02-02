// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"errors"
	"fmt"
)

// Sentinel errors for tool invocation.
// These errors can be used with errors.Is for detection.
var (
	// ErrUnknownTool indicates a tool call referenced an unknown tool.
	ErrUnknownTool = errors.New("unknown tool")

	// ErrInvalidArguments indicates the arguments could not be parsed or validated.
	ErrInvalidArguments = errors.New("invalid arguments")

	// ErrMaxIterations indicates the maximum iteration limit was exceeded.
	ErrMaxIterations = errors.New("max iterations exceeded")

	// ErrConsecutiveErrors indicates the maximum consecutive error limit was exceeded.
	ErrConsecutiveErrors = errors.New("max consecutive errors exceeded")

	// ErrInvocationDisabled indicates tool invocation is disabled.
	ErrInvocationDisabled = errors.New("tool invocation is disabled")

	// ErrToolTimeout indicates a tool invocation exceeded its timeout.
	ErrToolTimeout = errors.New("tool invocation timeout")
)

// InvocationError wraps errors from tool invocation with context.
// This error type preserves the original error while adding tool-specific context.
type InvocationError struct {
	// ToolName is the name of the tool that failed.
	ToolName string

	// Cause is the underlying error.
	Cause error
}

// Error returns a formatted error message including the tool name.
func (e *InvocationError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool %q invocation failed: %v", e.ToolName, e.Cause)
	}
	return fmt.Sprintf("tool %q invocation failed", e.ToolName)
}

// Unwrap returns the underlying cause for errors.Is and errors.As.
func (e *InvocationError) Unwrap() error {
	return e.Cause
}

// NewInvocationError creates an InvocationError for the given tool and cause.
func NewInvocationError(toolName string, cause error) *InvocationError {
	return &InvocationError{
		ToolName: toolName,
		Cause:    cause,
	}
}

// InvocationPanicError captures panic details from tool invocation.
// This error type is used when a tool panics during execution.
type InvocationPanicError struct {
	// ToolName is the name of the tool that panicked.
	ToolName string

	// Panic is the value recovered from the panic.
	Panic interface{}

	// Stack is the stack trace captured at the panic point.
	Stack []byte
}

// Error returns a formatted error message including the panic value.
func (e *InvocationPanicError) Error() string {
	return fmt.Sprintf("tool %q panicked: %v", e.ToolName, e.Panic)
}

// StackTrace returns the captured stack trace as a string.
func (e *InvocationPanicError) StackTrace() string {
	return string(e.Stack)
}

// NewInvocationPanicError creates an InvocationPanicError with the given details.
func NewInvocationPanicError(toolName string, panicValue interface{}, stack []byte) *InvocationPanicError {
	return &InvocationPanicError{
		ToolName: toolName,
		Panic:    panicValue,
		Stack:    stack,
	}
}

// UnknownToolError provides details about an unknown tool reference.
type UnknownToolError struct {
	// Name is the unknown tool name that was requested.
	Name string

	// AvailableTools lists the tools that are available.
	AvailableTools []string
}

// Error returns a formatted error message with the unknown tool name.
func (e *UnknownToolError) Error() string {
	return fmt.Sprintf("unknown tool: %q", e.Name)
}

// Is reports whether the target error is ErrUnknownTool.
func (e *UnknownToolError) Is(target error) bool {
	return target == ErrUnknownTool
}

// NewUnknownToolError creates an UnknownToolError with available tool names.
func NewUnknownToolError(name string, available []string) *UnknownToolError {
	return &UnknownToolError{
		Name:           name,
		AvailableTools: available,
	}
}

// ArgumentError provides details about argument validation failures.
type ArgumentError struct {
	// ToolName is the name of the tool with invalid arguments.
	ToolName string

	// Message describes the validation failure.
	Message string

	// Cause is the underlying parsing or validation error.
	Cause error
}

// Error returns a formatted error message.
func (e *ArgumentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("invalid arguments for tool %q: %s: %v", e.ToolName, e.Message, e.Cause)
	}
	return fmt.Sprintf("invalid arguments for tool %q: %s", e.ToolName, e.Message)
}

// Unwrap returns the underlying cause for errors.Is and errors.As.
func (e *ArgumentError) Unwrap() error {
	return e.Cause
}

// Is reports whether the target error is ErrInvalidArguments.
func (e *ArgumentError) Is(target error) bool {
	return target == ErrInvalidArguments
}

// NewArgumentError creates an ArgumentError with the given details.
func NewArgumentError(toolName, message string, cause error) *ArgumentError {
	return &ArgumentError{
		ToolName: toolName,
		Message:  message,
		Cause:    cause,
	}
}
