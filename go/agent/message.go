// Copyright (c) Microsoft. All rights reserved.

package agent

// Message represents a chat message in a conversation.
// This is a minimal stub that will be expanded in User Story 1.3.2.
// The full implementation will include Role, Contents, ToolCalls, and other fields.
type Message struct {
	// Role indicates who authored the message (system, user, assistant, tool).
	Role string

	// Content is the text content of the message.
	Content string
}
