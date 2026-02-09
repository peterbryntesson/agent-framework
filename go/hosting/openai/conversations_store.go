// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// SortOrder defines ordering for list operations.
type SortOrder string

const (
	SortOrderAsc  SortOrder = "asc"
	SortOrderDesc SortOrder = "desc"
)

// ConversationStore defines storage operations for conversations.
type ConversationStore interface {
	CreateConversation(ctx context.Context, conversation Conversation) (Conversation, error)
	GetConversation(ctx context.Context, conversationID string) (Conversation, bool, error)
	UpdateConversation(ctx context.Context, conversation Conversation) (Conversation, bool, error)
	DeleteConversation(ctx context.Context, conversationID string) (bool, error)
	AddItems(ctx context.Context, conversationID string, items []ItemResource) error
	GetItem(ctx context.Context, conversationID, itemID string) (ItemResource, bool, error)
	ListItems(ctx context.Context, conversationID string, limit int, order SortOrder, after string) (ListItemsResponse, error)
	DeleteItem(ctx context.Context, conversationID, itemID string) (bool, error)
}

// AgentConversationIndex tracks conversation IDs by agent.
type AgentConversationIndex interface {
	AddConversation(ctx context.Context, agentID, conversationID string) error
	RemoveConversation(ctx context.Context, agentID, conversationID string) error
	GetConversationIDs(ctx context.Context, agentID string) ([]string, error)
}

// InMemoryConversationStore is an in-memory conversation store.
type InMemoryConversationStore struct {
	mu            sync.Mutex
	conversations map[string]*conversationState
}

type conversationState struct {
	conversation Conversation
	items        []ItemResource
}

// NewInMemoryConversationStore creates a new in-memory store.
func NewInMemoryConversationStore() *InMemoryConversationStore {
	return &InMemoryConversationStore{
		conversations: make(map[string]*conversationState),
	}
}

// CreateConversation stores a new conversation.
func (s *InMemoryConversationStore) CreateConversation(ctx context.Context, conversation Conversation) (Conversation, error) {
	if ctx.Err() != nil {
		return Conversation{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.conversations[conversation.ID]; exists {
		return Conversation{}, errors.New("conversation already exists")
	}

	if conversation.CreatedAt == 0 {
		conversation.CreatedAt = time.Now().Unix()
	}
	if conversation.Object == "" {
		conversation.Object = "conversation"
	}

	s.conversations[conversation.ID] = &conversationState{conversation: conversation}
	return conversation, nil
}

// GetConversation returns a conversation by ID.
func (s *InMemoryConversationStore) GetConversation(ctx context.Context, conversationID string) (Conversation, bool, error) {
	if ctx.Err() != nil {
		return Conversation{}, false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.conversations[conversationID]
	if !ok {
		return Conversation{}, false, nil
	}
	return state.conversation, true, nil
}

// UpdateConversation updates conversation metadata.
func (s *InMemoryConversationStore) UpdateConversation(ctx context.Context, conversation Conversation) (Conversation, bool, error) {
	if ctx.Err() != nil {
		return Conversation{}, false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.conversations[conversation.ID]
	if !ok {
		return Conversation{}, false, nil
	}
	state.conversation = conversation
	return conversation, true, nil
}

// DeleteConversation deletes a conversation.
func (s *InMemoryConversationStore) DeleteConversation(ctx context.Context, conversationID string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.conversations[conversationID]; !ok {
		return false, nil
	}
	delete(s.conversations, conversationID)
	return true, nil
}

// AddItems adds items to a conversation.
func (s *InMemoryConversationStore) AddItems(ctx context.Context, conversationID string, items []ItemResource) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.conversations[conversationID]
	if !ok {
		return errors.New("conversation not found")
	}

	state.items = append(state.items, items...)
	return nil
}

// GetItem retrieves a specific item.
func (s *InMemoryConversationStore) GetItem(ctx context.Context, conversationID, itemID string) (ItemResource, bool, error) {
	if ctx.Err() != nil {
		return ItemResource{}, false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.conversations[conversationID]
	if !ok {
		return ItemResource{}, false, nil
	}

	for _, item := range state.items {
		if item.ID == itemID {
			return item, true, nil
		}
	}
	return ItemResource{}, false, nil
}

// ListItems lists items with basic pagination.
func (s *InMemoryConversationStore) ListItems(ctx context.Context, conversationID string, limit int, order SortOrder, after string) (ListItemsResponse, error) {
	if ctx.Err() != nil {
		return ListItemsResponse{}, ctx.Err()
	}

	if limit <= 0 {
		limit = 20
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.conversations[conversationID]
	if !ok {
		return ListItemsResponse{}, errors.New("conversation not found")
	}

	items := append([]ItemResource(nil), state.items...)
	if order == SortOrderDesc {
		sort.SliceStable(items, func(i, j int) bool { return items[i].CreatedAt > items[j].CreatedAt })
	}

	start := 0
	if after != "" {
		for i, item := range items {
			if item.ID == after {
				start = i + 1
				break
			}
		}
	}

	selected := items
	if start < len(items) {
		selected = items[start:]
	} else {
		selected = []ItemResource{}
	}

	hasMore := false
	if len(selected) > limit {
		hasMore = true
		selected = selected[:limit]
	}

	resp := ListItemsResponse{
		Object:  "list",
		Data:    selected,
		HasMore: hasMore,
	}
	if len(selected) > 0 {
		resp.FirstID = selected[0].ID
		resp.LastID = selected[len(selected)-1].ID
	}
	return resp, nil
}

// DeleteItem deletes a specific item.
func (s *InMemoryConversationStore) DeleteItem(ctx context.Context, conversationID, itemID string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.conversations[conversationID]
	if !ok {
		return false, nil
	}

	for i, item := range state.items {
		if item.ID == itemID {
			state.items = append(state.items[:i], state.items[i+1:]...)
			return true, nil
		}
	}
	return false, nil
}

// InMemoryAgentConversationIndex tracks conversations by agent ID.
type InMemoryAgentConversationIndex struct {
	mu    sync.Mutex
	index map[string][]string
}

// NewInMemoryAgentConversationIndex creates a new index.
func NewInMemoryAgentConversationIndex() *InMemoryAgentConversationIndex {
	return &InMemoryAgentConversationIndex{index: make(map[string][]string)}
}

// AddConversation adds a conversation to the index.
func (i *InMemoryAgentConversationIndex) AddConversation(ctx context.Context, agentID, conversationID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	i.index[agentID] = append(i.index[agentID], conversationID)
	return nil
}

// RemoveConversation removes a conversation from the index.
func (i *InMemoryAgentConversationIndex) RemoveConversation(ctx context.Context, agentID, conversationID string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	convs := i.index[agentID]
	for idx, id := range convs {
		if id == conversationID {
			i.index[agentID] = append(convs[:idx], convs[idx+1:]...)
			break
		}
	}
	return nil
}

// GetConversationIDs returns conversation IDs for an agent.
func (i *InMemoryAgentConversationIndex) GetConversationIDs(ctx context.Context, agentID string) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	ids := append([]string(nil), i.index[agentID]...)
	return ids, nil
}
