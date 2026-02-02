// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

// Message is an alias to chat.Message for use in agent packages.
// This provides type compatibility between the agent and chat packages.
type Message = chat.Message

// NewUserMessage creates a new user message with the given text content.
// This is a convenience wrapper around chat.NewUserMessage.
func NewUserMessage(text string) Message {
	return chat.NewUserMessage(text)
}

// NewSystemMessage creates a new system message with the given text content.
// This is a convenience wrapper around chat.NewSystemMessage.
func NewSystemMessage(text string) Message {
	return chat.NewSystemMessage(text)
}

// NewAssistantMessage creates a new assistant message with the given text content.
// This is a convenience wrapper around chat.NewAssistantMessage.
func NewAssistantMessage(text string) Message {
	return chat.NewAssistantMessage(text)
}

// NewToolMessage creates a new tool message with the given result content.
// This is a convenience wrapper around chat.NewToolMessage.
func NewToolMessage(toolCallID, content string) Message {
	return chat.NewToolMessage(toolCallID, content)
}

// NewAssistantMessageWithToolCalls creates a new assistant message with tool calls.
// This is a convenience wrapper around chat.NewAssistantMessageWithToolCalls.
func NewAssistantMessageWithToolCalls(toolCalls []chat.ToolCall) Message {
	return chat.NewAssistantMessageWithToolCalls(toolCalls)
}

// NewMessageWithContents creates a new message with the given role and content items.
// This is a convenience wrapper around chat.NewMessageWithContents.
func NewMessageWithContents(role chat.Role, contents ...chat.Content) Message {
	return chat.NewMessageWithContents(role, contents...)
}

// SimpleMessage represents a simplified message with just role and content text.
// Use this for scenarios where the full chat.Message structure is not needed.
type SimpleMessage struct {
	// Role indicates who authored the message (system, user, assistant, tool).
	Role string

	// Content is the text content of the message.
	Content string

	// CreatedAt is the timestamp when the message was created.
	CreatedAt time.Time
}

// ToMessage converts a SimpleMessage to a full Message.
func (m *SimpleMessage) ToMessage() Message {
	role := chat.Role(m.Role)
	createdAt := m.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	return chat.Message{
		Role:      role,
		Contents:  []chat.Content{chat.NewTextContent(m.Content)},
		CreatedAt: createdAt,
	}
}
