// Copyright (c) Microsoft. All rights reserved.

package chat

import (
	"strings"
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
// Messages can contain multiple content types including text, images, tool calls, and tool results.
type Message struct {
	// Role indicates who authored the message.
	Role Role

	// Contents is the structured content of the message.
	// A message can contain multiple content items (e.g., text and images).
	// Use the Text() method to extract concatenated text content.
	Contents []Content

	// Name is an optional name for the message author.
	// Used for multi-participant conversations or tool identification.
	Name string

	// ToolCalls contains tool/function calls requested by the assistant.
	// Only populated for messages with Role == RoleAssistant.
	ToolCalls []ToolCall

	// ToolCallID identifies the tool call this message responds to.
	// Only used for messages with Role == RoleTool.
	ToolCallID string

	// CreatedAt is the timestamp when the message was created.
	CreatedAt time.Time

	// RawRepresentation holds the original provider-specific message data.
	// This enables access to provider-specific fields not in the common schema.
	RawRepresentation interface{}
}

// Text returns the concatenated text content from all TextContent items in the message.
// Returns an empty string if the message contains no text content.
func (m *Message) Text() string {
	if m == nil || len(m.Contents) == 0 {
		return ""
	}

	var texts []string
	for _, c := range m.Contents {
		if tc, ok := c.(*TextContent); ok {
			texts = append(texts, tc.Text)
		}
	}
	return strings.Join(texts, "")
}

// NewUserMessage creates a new user message with the given text content.
func NewUserMessage(text string) Message {
	return Message{
		Role:      RoleUser,
		Contents:  []Content{NewTextContent(text)},
		CreatedAt: time.Now(),
	}
}

// NewSystemMessage creates a new system message with the given text content.
func NewSystemMessage(text string) Message {
	return Message{
		Role:      RoleSystem,
		Contents:  []Content{NewTextContent(text)},
		CreatedAt: time.Now(),
	}
}

// NewAssistantMessage creates a new assistant message with the given text content.
func NewAssistantMessage(text string) Message {
	return Message{
		Role:      RoleAssistant,
		Contents:  []Content{NewTextContent(text)},
		CreatedAt: time.Now(),
	}
}

// NewToolMessage creates a new tool message with the given result content.
func NewToolMessage(toolCallID, content string) Message {
	return Message{
		Role:       RoleTool,
		Contents:   []Content{NewTextContent(content)},
		ToolCallID: toolCallID,
		CreatedAt:  time.Now(),
	}
}

// NewAssistantMessageWithToolCalls creates a new assistant message with tool calls.
func NewAssistantMessageWithToolCalls(toolCalls []ToolCall) Message {
	return Message{
		Role:      RoleAssistant,
		ToolCalls: toolCalls,
		CreatedAt: time.Now(),
	}
}

// NewMessageWithContents creates a new message with the given role and content items.
func NewMessageWithContents(role Role, contents ...Content) Message {
	return Message{
		Role:      role,
		Contents:  contents,
		CreatedAt: time.Now(),
	}
}
