// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"encoding/json"
)

// Conversation represents a stored conversation.
type Conversation struct {
	ID        string            `json:"id"`
	Object    string            `json:"object"`
	CreatedAt int64             `json:"created_at"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// ConversationReference references a conversation by ID.
type ConversationReference struct {
	ID       string            `json:"id"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UnmarshalJSON supports conversation references as string or object.
func (c *ConversationReference) UnmarshalJSON(data []byte) error {
	var id string
	if err := json.Unmarshal(data, &id); err == nil {
		c.ID = id
		return nil
	}

	var tmp struct {
		ID       string            `json:"id"`
		Metadata map[string]string `json:"metadata,omitempty"`
	}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	c.ID = tmp.ID
	c.Metadata = tmp.Metadata
	return nil
}

// CreateConversationRequest represents a create conversation request.
type CreateConversationRequest struct {
	Items    []ItemParam       `json:"items,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// UpdateConversationRequest represents a conversation update request.
type UpdateConversationRequest struct {
	Metadata map[string]string `json:"metadata"`
}

// CreateItemsRequest represents a request to add items to a conversation.
type CreateItemsRequest struct {
	Items []ItemParam `json:"items"`
}

// ListConversationsResponse represents a list response for conversations.
type ListConversationsResponse struct {
	Data    []Conversation `json:"data"`
	HasMore bool           `json:"has_more"`
}
