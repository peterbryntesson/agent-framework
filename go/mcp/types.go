// Copyright (c) Microsoft. All rights reserved.

package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ProtocolVersion is the MCP protocol version supported by this implementation.
const ProtocolVersion = "2024-11-05"

// JSON-RPC constants.
const (
	// JSONRPCVersion is the JSON-RPC version used by MCP.
	JSONRPCVersion = "2.0"
)

// ToolInfo describes an MCP tool.
// Tools are executable functions that can be invoked by clients.
type ToolInfo struct {
	// Name is the unique identifier for this tool.
	Name string `json:"name"`

	// Description is a human-readable description of what the tool does.
	Description string `json:"description,omitempty"`

	// InputSchema is the JSON Schema describing the tool's input parameters.
	InputSchema json.RawMessage `json:"inputSchema,omitempty"`
}

// Content represents MCP content returned from tool invocations or resources.
type Content struct {
	// Type specifies the content type: "text", "image", or "resource".
	Type string `json:"type"`

	// Text contains text content (when Type is "text").
	Text string `json:"text,omitempty"`

	// Data contains base64-encoded binary data (when Type is "image").
	Data string `json:"data,omitempty"`

	// MimeType specifies the MIME type for binary data.
	MimeType string `json:"mimeType,omitempty"`

	// URI is a resource URI (when Type is "resource").
	URI string `json:"uri,omitempty"`
}

// ContentType constants for Content.Type.
const (
	ContentTypeText     = "text"
	ContentTypeImage    = "image"
	ContentTypeResource = "resource"
)

// NewTextContent creates a text content item.
func NewTextContent(text string) Content {
	return Content{Type: ContentTypeText, Text: text}
}

// NewImageContent creates an image content item with base64-encoded data.
func NewImageContent(data string, mimeType string) Content {
	return Content{Type: ContentTypeImage, Data: data, MimeType: mimeType}
}

// CallToolResult is the result of calling an MCP tool.
type CallToolResult struct {
	// Content contains the result content items.
	Content []Content `json:"content"`

	// IsError indicates if the tool call resulted in an error.
	IsError bool `json:"isError,omitempty"`
}

// ResourceInfo describes an MCP resource.
// Resources are data items that can be read by clients.
type ResourceInfo struct {
	// URI is the unique identifier for this resource.
	URI string `json:"uri"`

	// Name is a human-readable name for the resource.
	Name string `json:"name"`

	// Description is an optional description of the resource.
	Description string `json:"description,omitempty"`

	// MimeType is the MIME type of the resource content.
	MimeType string `json:"mimeType,omitempty"`
}

// ResourceContent is the content of a resource.
type ResourceContent struct {
	// URI is the resource URI.
	URI string `json:"uri"`

	// MimeType is the MIME type of the content.
	MimeType string `json:"mimeType,omitempty"`

	// Text contains text content.
	Text string `json:"text,omitempty"`

	// Blob contains base64-encoded binary content.
	Blob string `json:"blob,omitempty"`
}

// PromptInfo describes an MCP prompt template.
type PromptInfo struct {
	// Name is the unique identifier for this prompt.
	Name string `json:"name"`

	// Description is a human-readable description of the prompt.
	Description string `json:"description,omitempty"`

	// Arguments describes the prompt's arguments.
	Arguments []PromptArgument `json:"arguments,omitempty"`
}

// PromptArgument describes a prompt template argument.
type PromptArgument struct {
	// Name is the argument name.
	Name string `json:"name"`

	// Description is a human-readable description.
	Description string `json:"description,omitempty"`

	// Required indicates if this argument must be provided.
	Required bool `json:"required,omitempty"`
}

// PromptMessage describes a prompt message template.
type PromptMessage struct {
	// Role is the chat role for the message.
	Role string `json:"role"`

	// Content is the message content.
	Content []Content `json:"content"`
}

// Prompt describes a prompt template with messages.
type Prompt struct {
	// Name is the unique identifier for this prompt.
	Name string `json:"name"`

	// Description is a human-readable description of the prompt.
	Description string `json:"description,omitempty"`

	// Arguments describes the prompt's arguments.
	Arguments []PromptArgument `json:"arguments,omitempty"`

	// Messages defines the prompt template messages.
	Messages []PromptMessage `json:"messages,omitempty"`
}

// Implementation describes the client or server implementation.
type Implementation struct {
	// Name is the implementation name.
	Name string `json:"name"`

	// Version is the implementation version.
	Version string `json:"version"`
}

// ClientCapabilities describes client capabilities during initialization.
type ClientCapabilities struct {
	// Roots indicates capability to provide filesystem roots.
	Roots *RootsCapability `json:"roots,omitempty"`

	// Sampling indicates capability to perform LLM sampling.
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

// RootsCapability describes filesystem roots capability.
type RootsCapability struct {
	// ListChanged indicates support for roots/list_changed notifications.
	ListChanged bool `json:"listChanged,omitempty"`
}

// SamplingCapability describes LLM sampling capability.
type SamplingCapability struct{}

// ServerCapabilities describes server capabilities during initialization.
type ServerCapabilities struct {
	// Prompts indicates prompt template support.
	Prompts *PromptsCapability `json:"prompts,omitempty"`

	// Resources indicates resource support.
	Resources *ResourcesCapability `json:"resources,omitempty"`

	// Tools indicates tool support.
	Tools *ToolsCapability `json:"tools,omitempty"`

	// Logging indicates logging support.
	Logging *LoggingCapability `json:"logging,omitempty"`
}

// PromptsCapability describes prompt capabilities.
type PromptsCapability struct {
	// ListChanged indicates support for prompts/list_changed notifications.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ResourcesCapability describes resource capabilities.
type ResourcesCapability struct {
	// Subscribe indicates support for resource subscriptions.
	Subscribe bool `json:"subscribe,omitempty"`

	// ListChanged indicates support for resources/list_changed notifications.
	ListChanged bool `json:"listChanged,omitempty"`
}

// ToolsCapability describes tool capabilities.
type ToolsCapability struct {
	// ListChanged indicates support for tools/list_changed notifications.
	ListChanged bool `json:"listChanged,omitempty"`
}

// LoggingCapability describes logging capabilities.
type LoggingCapability struct{}

// LoggingLevel defines the logging level for MCP logging messages.
type LoggingLevel string

// LoggingLevel constants.
const (
	LoggingLevelDebug     LoggingLevel = "debug"
	LoggingLevelInfo      LoggingLevel = "info"
	LoggingLevelNotice    LoggingLevel = "notice"
	LoggingLevelWarning   LoggingLevel = "warning"
	LoggingLevelError     LoggingLevel = "error"
	LoggingLevelCritical  LoggingLevel = "critical"
	LoggingLevelAlert     LoggingLevel = "alert"
	LoggingLevelEmergency LoggingLevel = "emergency"
)

// LoggingMessageNotificationParams describes a logging notification.
type LoggingMessageNotificationParams struct {
	// Level is the logging severity.
	Level LoggingLevel `json:"level"`

	// Data is the logging payload.
	Data json.RawMessage `json:"data,omitempty"`
}

// LoggingSetLevelParams describes a request to set server logging level.
type LoggingSetLevelParams struct {
	// Level is the desired logging level.
	Level LoggingLevel `json:"level"`
}

// InitializeParams contains initialization parameters from the client.
type InitializeParams struct {
	// ProtocolVersion is the protocol version the client supports.
	ProtocolVersion string `json:"protocolVersion"`

	// Capabilities describes client capabilities.
	Capabilities ClientCapabilities `json:"capabilities"`

	// ClientInfo describes the client implementation.
	ClientInfo Implementation `json:"clientInfo"`
}

// InitializeResult is the server's response to initialization.
type InitializeResult struct {
	// ProtocolVersion is the protocol version the server will use.
	ProtocolVersion string `json:"protocolVersion"`

	// Capabilities describes server capabilities.
	Capabilities ServerCapabilities `json:"capabilities"`

	// ServerInfo describes the server implementation.
	ServerInfo Implementation `json:"serverInfo"`

	// Instructions provides optional usage instructions.
	Instructions string `json:"instructions,omitempty"`
}

// JSON-RPC types for internal protocol communication.

// Request represents a JSON-RPC request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *ResponseError  `json:"error,omitempty"`
}

// ResponseError is a JSON-RPC error object.
type ResponseError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Error implements the error interface.
func (e *ResponseError) Error() string {
	return fmt.Sprintf("MCP error %d: %s", e.Code, e.Message)
}

// Standard JSON-RPC error codes.
const (
	// ParseError indicates invalid JSON.
	ParseError = -32700

	// InvalidRequest indicates the JSON is not a valid request.
	InvalidRequest = -32600

	// MethodNotFound indicates the method does not exist.
	MethodNotFound = -32601

	// InvalidParams indicates invalid method parameters.
	InvalidParams = -32602

	// InternalError indicates an internal JSON-RPC error.
	InternalError = -32603
)

// Notification represents a JSON-RPC notification (no response expected).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCP-specific request/response types.

// ListToolsResult is the result of tools/list.
type ListToolsResult struct {
	Tools      []ToolInfo `json:"tools"`
	NextCursor string     `json:"nextCursor,omitempty"`
}

// CallToolParams are the parameters for tools/call.
type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ListResourcesResult is the result of resources/list.
type ListResourcesResult struct {
	Resources  []ResourceInfo `json:"resources"`
	NextCursor string         `json:"nextCursor,omitempty"`
}

// ReadResourceParams are the parameters for resources/read.
type ReadResourceParams struct {
	URI string `json:"uri"`
}

// ReadResourceResult is the result of resources/read.
type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// ListPromptsResult is the result of prompts/list.
type ListPromptsResult struct {
	Prompts    []PromptInfo `json:"prompts"`
	NextCursor string       `json:"nextCursor,omitempty"`
}

// GetPromptParams are the parameters for prompts/get.
type GetPromptParams struct {
	// Name is the prompt identifier.
	Name string `json:"name"`

	// Arguments are prompt arguments supplied by the caller.
	Arguments map[string]string `json:"arguments,omitempty"`
}

// GetPromptResult is the result of prompts/get.
type GetPromptResult struct {
	Prompt Prompt `json:"prompt"`
}

// SamplingMessage describes a message for sampling.
type SamplingMessage struct {
	// Role is the role of the message.
	Role string `json:"role"`

	// Content contains the message content.
	Content []Content `json:"content"`
}

// CreateMessageParams are the parameters for sampling/createMessage.
type CreateMessageParams struct {
	// Messages are the input messages for sampling.
	Messages []SamplingMessage `json:"messages"`

	// Model is the requested model identifier.
	Model string `json:"model,omitempty"`

	// Temperature controls response randomness.
	Temperature *float64 `json:"temperature,omitempty"`

	// MaxTokens limits the response token count.
	MaxTokens *int `json:"maxTokens,omitempty"`

	// StopSequences terminate generation when matched.
	StopSequences []string `json:"stopSequences,omitempty"`
}

// CreateMessageResult is the result of sampling/createMessage.
type CreateMessageResult struct {
	// Role is the role of the generated message.
	Role string `json:"role"`

	// Content is the generated content.
	Content Content `json:"content"`

	// Model is the model that produced the response.
	Model string `json:"model,omitempty"`
}

// Sentinel errors for MCP operations.
var (
	// ErrNotInitialized indicates the client has not completed initialization.
	ErrNotInitialized = errors.New("mcp: client not initialized")

	// ErrClosed indicates the client or server has been closed.
	ErrClosed = errors.New("mcp: connection closed")

	// ErrTimeout indicates an operation timed out.
	ErrTimeout = errors.New("mcp: operation timed out")

	// ErrToolNotFound indicates the requested tool was not found.
	ErrToolNotFound = errors.New("mcp: tool not found")

	// ErrResourceNotFound indicates the requested resource was not found.
	ErrResourceNotFound = errors.New("mcp: resource not found")

	// ErrPromptNotFound indicates the requested prompt was not found.
	ErrPromptNotFound = errors.New("mcp: prompt not found")

	// ErrInvalidParams indicates invalid parameters were provided.
	ErrInvalidParams = errors.New("mcp: invalid parameters")

	// ErrProtocolVersion indicates a protocol version mismatch.
	ErrProtocolVersion = errors.New("mcp: unsupported protocol version")
)

// MCPError wraps errors from MCP operations with context.
type MCPError struct {
	// Operation is the MCP operation that failed.
	Operation string

	// Cause is the underlying error.
	Cause error
}

// Error returns a formatted error message.
func (e *MCPError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("mcp %s: %v", e.Operation, e.Cause)
	}
	return fmt.Sprintf("mcp %s failed", e.Operation)
}

// Unwrap returns the underlying cause for errors.Is and errors.As.
func (e *MCPError) Unwrap() error {
	return e.Cause
}

// NewMCPError creates an MCPError for the given operation and cause.
func NewMCPError(operation string, cause error) *MCPError {
	return &MCPError{
		Operation: operation,
		Cause:     cause,
	}
}
