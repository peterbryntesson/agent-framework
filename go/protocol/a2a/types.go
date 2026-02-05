// Copyright (c) Microsoft. All rights reserved.

package a2a

import "time"

// AgentCard describes an agent's capabilities for discovery.
// This is the primary means by which agents advertise their
// capabilities to other agents in the A2A protocol.
type AgentCard struct {
	// Name is the human-readable name of the agent
	Name string `json:"name"`

	// Description explains the agent's purpose
	Description string `json:"description,omitempty"`

	// URL is the base URL for the agent's A2A endpoints
	URL string `json:"url"`

	// Provider is the organization providing the agent
	Provider *AgentProvider `json:"provider,omitempty"`

	// Version is the agent's version string
	Version string `json:"version,omitempty"`

	// Capabilities describes what the agent can do
	Capabilities *AgentCapabilities `json:"capabilities,omitempty"`

	// Authentication describes how to authenticate
	Authentication *AgentAuthentication `json:"authentication,omitempty"`

	// Skills lists specific capabilities
	Skills []AgentSkill `json:"skills,omitempty"`
}

// AgentProvider identifies who provides the agent.
type AgentProvider struct {
	// Name is the provider's name
	Name string `json:"name"`

	// URL is the provider's website
	URL string `json:"url,omitempty"`
}

// AgentCapabilities describes supported features.
type AgentCapabilities struct {
	// Streaming indicates SSE support
	Streaming bool `json:"streaming,omitempty"`

	// PushNotifications indicates webhook support
	PushNotifications bool `json:"push_notifications,omitempty"`

	// StateTransitions indicates supported state changes
	StateTransitions []string `json:"state_transitions,omitempty"`
}

// AgentAuthentication describes authentication requirements.
type AgentAuthentication struct {
	// Schemes lists supported auth schemes (e.g., "bearer", "api_key")
	Schemes []string `json:"schemes,omitempty"`
}

// AgentSkill describes a specific agent capability.
type AgentSkill struct {
	// ID is the unique identifier for the skill
	ID string `json:"id"`

	// Name is the human-readable name
	Name string `json:"name"`

	// Description explains what the skill does
	Description string `json:"description,omitempty"`

	// Tags categorize the skill
	Tags []string `json:"tags,omitempty"`
}

// TaskState represents the current state of a task.
type TaskState string

const (
	// TaskStatePending indicates the task is waiting to be processed.
	TaskStatePending TaskState = "pending"

	// TaskStateRunning indicates the task is currently being processed.
	TaskStateRunning TaskState = "running"

	// TaskStateCompleted indicates the task finished successfully.
	TaskStateCompleted TaskState = "completed"

	// TaskStateFailed indicates the task failed with an error.
	TaskStateFailed TaskState = "failed"

	// TaskStateCancelled indicates the task was canceled.
	TaskStateCancelled TaskState = "cancelled"
)

// Task represents a unit of work in the A2A protocol.
// Tasks are the primary way to track ongoing work with an agent.
type Task struct {
	// ID is the unique task identifier
	ID string `json:"id"`

	// ContextID groups related tasks (conversation)
	ContextID string `json:"context_id,omitempty"`

	// State is the current task state
	State TaskState `json:"state"`

	// Messages contains the conversation history
	Messages []Message `json:"messages,omitempty"`

	// Artifacts contains generated outputs
	Artifacts []Artifact `json:"artifacts,omitempty"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the task was created
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the task was last modified
	UpdatedAt time.Time `json:"updated_at"`

	// Error contains error details if failed
	Error *TaskError `json:"error,omitempty"`
}

// TaskError describes why a task failed.
type TaskError struct {
	// Code is a machine-readable error code
	Code string `json:"code"`

	// Message is a human-readable error description
	Message string `json:"message"`
}

// CreateTaskRequest contains parameters for creating a task.
type CreateTaskRequest struct {
	// ContextID optionally groups this task with related tasks
	ContextID string `json:"context_id,omitempty"`

	// Message is the initial message to start the task
	Message *Message `json:"message"`

	// Metadata contains additional properties for the task
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MessageRole indicates the sender of a message.
type MessageRole string

const (
	// RoleUser indicates a message from the human user.
	RoleUser MessageRole = "user"

	// RoleAssistant indicates a message from the AI assistant.
	RoleAssistant MessageRole = "assistant"

	// RoleSystem indicates a system message.
	RoleSystem MessageRole = "system"
)

// Message represents a message in the A2A protocol.
// Messages contain one or more parts that can be text, data, or files.
type Message struct {
	// Role indicates who sent the message
	Role MessageRole `json:"role"`

	// Parts contains the message content
	Parts []Part `json:"parts"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the message was created
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// Text returns the concatenated text content from all text parts.
func (m *Message) Text() string {
	var text string
	for _, part := range m.Parts {
		if part.Type == PartTypeText {
			text += part.Text
		}
	}
	return text
}

// PartType indicates the type of message part.
type PartType string

const (
	// PartTypeText indicates a text content part.
	PartTypeText PartType = "text"

	// PartTypeData indicates a structured data part.
	PartTypeData PartType = "data"

	// PartTypeFile indicates a file reference part.
	PartTypeFile PartType = "file"
)

// Part is a component of a message (text, data, file, etc.).
type Part struct {
	// Type indicates the part type
	Type PartType `json:"type"`

	// Text contains text content (for text parts)
	Text string `json:"text,omitempty"`

	// Data contains structured data (for data parts)
	Data interface{} `json:"data,omitempty"`

	// MimeType indicates the content type
	MimeType string `json:"mime_type,omitempty"`

	// URI points to external content
	URI string `json:"uri,omitempty"`
}

// NewTextPart creates a text message part.
func NewTextPart(text string) Part {
	return Part{Type: PartTypeText, Text: text}
}

// NewDataPart creates a data message part.
func NewDataPart(data interface{}, mimeType string) Part {
	return Part{Type: PartTypeData, Data: data, MimeType: mimeType}
}

// NewFilePart creates a file reference part.
func NewFilePart(uri, mimeType string) Part {
	return Part{Type: PartTypeFile, URI: uri, MimeType: mimeType}
}

// NewUserMessage creates a user message with text content.
func NewUserMessage(text string) Message {
	return Message{
		Role:      RoleUser,
		Parts:     []Part{NewTextPart(text)},
		CreatedAt: time.Now(),
	}
}

// NewAssistantMessage creates an assistant message with text content.
func NewAssistantMessage(text string) Message {
	return Message{
		Role:      RoleAssistant,
		Parts:     []Part{NewTextPart(text)},
		CreatedAt: time.Now(),
	}
}

// Artifact represents an output generated by an agent.
// Artifacts are used to return structured outputs like generated
// documents, code, or other files.
type Artifact struct {
	// ID is the unique artifact identifier
	ID string `json:"id"`

	// Name is a human-readable name
	Name string `json:"name,omitempty"`

	// Description explains the artifact
	Description string `json:"description,omitempty"`

	// Parts contains the artifact content
	Parts []Part `json:"parts"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CreatedAt is when the artifact was created
	CreatedAt time.Time `json:"created_at"`
}

// NewArtifact creates a new artifact with the given ID and parts.
func NewArtifact(id string, parts ...Part) Artifact {
	return Artifact{
		ID:        id,
		Parts:     parts,
		CreatedAt: time.Now(),
	}
}

// SendMessageRequest contains parameters for sending a message to a task.
type SendMessageRequest struct {
	// Message is the message to send
	Message *Message `json:"message"`

	// Metadata contains additional properties
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// StreamEvent represents an event from a streaming response.
type StreamEvent struct {
	// Type indicates the event type
	Type string `json:"type"`

	// Data contains the raw event data
	Data interface{} `json:"data,omitempty"`

	// Message contains a message if this is a message event
	Message *Message `json:"message,omitempty"`

	// Task contains a task if this is a task event
	Task *Task `json:"task,omitempty"`

	// Error contains any error that occurred during streaming
	Error error `json:"-"`
}

// StreamEventType constants for event types.
const (
	StreamEventTypeMessage  = "message"
	StreamEventTypeTask     = "task"
	StreamEventTypeError    = "error"
	StreamEventTypeDone     = "done"
	StreamEventTypeProgress = "progress"
)
