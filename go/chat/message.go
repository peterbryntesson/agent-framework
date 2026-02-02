// Copyright (c) Microsoft. All rights reserved.

package chat

import (
	"time"
)

// Role represents the role of a message author in a conversation.
type Role string

const (
	// RoleSystem indicates a system prompt that sets the agent's behavior.
	RoleSystem Role = "system"

	// RoleUser indicates a message from the human user.
	RoleUser Role = "user"

	// RoleAssistant indicates a message from the AI assistant.
	RoleAssistant Role = "assistant"

	// RoleTool indicates a message containing tool/function results.
	RoleTool Role = "tool"
)

// Message represents a single message in a chat conversation.
// This is a minimal implementation for User Story 1.3.1.
// The full implementation with Content types will be added in User Story 1.3.2.
type Message struct {
	// Role indicates who authored the message.
	Role Role

	// Content is the text content of the message.
	// This is a simplified field; User Story 1.3.2 will add structured Content types.
	Content string

	// Name is an optional name for the message author.
	// Used for multi-participant conversations or tool identification.
	Name string

	// ToolCallID identifies the tool call this message responds to.
	// Only used for messages with Role == RoleTool.
	ToolCallID string

	// CreatedAt is the timestamp when the message was created.
	CreatedAt time.Time

	// RawRepresentation holds the original provider-specific message data.
	// This enables access to provider-specific fields not in the common schema.
	RawRepresentation interface{}
}

// NewUserMessage creates a new user message with the given text content.
func NewUserMessage(content string) Message {
	return Message{
		Role:      RoleUser,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// NewSystemMessage creates a new system message with the given text content.
func NewSystemMessage(content string) Message {
	return Message{
		Role:      RoleSystem,
		Content:   content,
		CreatedAt: time.Now(),
	}
}

// NewAssistantMessage creates a new assistant message with the given text content.
func NewAssistantMessage(content string) Message {
	return Message{
		Role:      RoleAssistant,
		Content:   content,
		CreatedAt: time.Now(),
	}
}
