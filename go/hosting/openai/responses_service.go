// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// ResponsesService defines operations for OpenAI responses.
type ResponsesService interface {
	CreateResponse(ctx context.Context, request CreateResponse) (Response, error)
	CreateResponseStreaming(ctx context.Context, request CreateResponse) (<-chan StreamingResponseEvent, Response, error)
	GetResponse(ctx context.Context, responseID string) (Response, bool, error)
	GetResponseStreaming(ctx context.Context, responseID string, startingAfter int) (<-chan StreamingResponseEvent, error)
	CancelResponse(ctx context.Context, responseID string) (Response, error)
	DeleteResponse(ctx context.Context, responseID string) (bool, error)
	ListResponseInputItems(ctx context.Context, responseID string, limit int, order SortOrder, after string) (ListItemsResponse, error)
}

// InMemoryResponsesService stores responses in memory.
type InMemoryResponsesService struct {
	agent             agent.Agent
	conversationStore ConversationStore

	mu        sync.Mutex
	responses map[string]*responseState
}

type responseState struct {
	response   Response
	inputItems []ItemResource
	events     []StreamingResponseEvent
	done       bool
	cond       *sync.Cond
}

// NewInMemoryResponsesService creates a new responses service.
func NewInMemoryResponsesService(ag agent.Agent, store ConversationStore) *InMemoryResponsesService {
	return &InMemoryResponsesService{
		agent:             ag,
		conversationStore: store,
		responses:         make(map[string]*responseState),
	}
}

// CreateResponse handles non-streaming response creation.
func (s *InMemoryResponsesService) CreateResponse(ctx context.Context, request CreateResponse) (Response, error) {
	responseID := s.newResponseID()
	state := s.initResponse(responseID, request)

	messages, inputItems, err := s.parseInput(request)
	if err != nil {
		return Response{}, err
	}
	state.inputItems = inputItems

	response, err := s.runAgent(ctx, state, request, messages)
	if err != nil {
		return Response{}, err
	}

	s.mu.Lock()
	state.response = response
	state.done = true
	state.cond.Broadcast()
	s.mu.Unlock()

	return response, nil
}

// CreateResponseStreaming handles streaming response creation.
func (s *InMemoryResponsesService) CreateResponseStreaming(ctx context.Context, request CreateResponse) (<-chan StreamingResponseEvent, Response, error) {
	responseID := s.newResponseID()
	state := s.initResponse(responseID, request)

	messages, inputItems, err := s.parseInput(request)
	if err != nil {
		return nil, Response{}, err
	}
	state.inputItems = inputItems

	ch := make(chan StreamingResponseEvent, 128)

	go func() {
		defer close(ch)
		s.mu.Lock()
		initial := append([]StreamingResponseEvent(nil), state.events...)
		s.mu.Unlock()
		for _, evt := range initial {
			ch <- evt
		}
		s.streamAgent(ctx, ch, state, request, messages)
	}()

	return ch, state.response, nil
}

// GetResponse retrieves a response.
func (s *InMemoryResponsesService) GetResponse(ctx context.Context, responseID string) (Response, bool, error) {
	if ctx.Err() != nil {
		return Response{}, false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.responses[responseID]
	if !ok {
		return Response{}, false, nil
	}
	return state.response, true, nil
}

// GetResponseStreaming streams stored events for a response.
func (s *InMemoryResponsesService) GetResponseStreaming(ctx context.Context, responseID string, startingAfter int) (<-chan StreamingResponseEvent, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	s.mu.Lock()
	state, ok := s.responses[responseID]
	if !ok {
		s.mu.Unlock()
		return nil, errors.New("response not found")
	}
	ch := make(chan StreamingResponseEvent, 128)
	cond := state.cond
	s.mu.Unlock()

	go func() {
		defer close(ch)
		index := startingAfter
		for {
			s.mu.Lock()
			for index >= len(state.events) && !state.done {
				cond.Wait()
			}

			if index < len(state.events) {
				batch := append([]StreamingResponseEvent(nil), state.events[index:]...)
				index = len(state.events)
				s.mu.Unlock()
				for _, evt := range batch {
					select {
					case <-ctx.Done():
						return
					case ch <- evt:
					}
				}
				continue
			}

			done := state.done
			s.mu.Unlock()
			if done {
				return
			}
		}
	}()

	return ch, nil
}

// CancelResponse cancels a response.
func (s *InMemoryResponsesService) CancelResponse(ctx context.Context, responseID string) (Response, error) {
	if ctx.Err() != nil {
		return Response{}, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state, ok := s.responses[responseID]
	if !ok {
		return Response{}, errors.New("response not found")
	}

	if state.response.Status == ResponseStatusCompleted || state.response.Status == ResponseStatusFailed {
		return Response{}, errors.New("response already completed")
	}

	state.response.Status = ResponseStatusCancelled
	state.done = true
	s.appendEventLocked(state, StreamingResponseEvent{
		Type:           "response.cancelled",
		SequenceNumber: s.nextSequence(state),
		Response:       &state.response,
	})

	state.cond.Broadcast()
	return state.response, nil
}

// DeleteResponse deletes a response.
func (s *InMemoryResponsesService) DeleteResponse(ctx context.Context, responseID string) (bool, error) {
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.responses[responseID]; !ok {
		return false, nil
	}
	delete(s.responses, responseID)
	return true, nil
}

// ListResponseInputItems lists input items for a response.
func (s *InMemoryResponsesService) ListResponseInputItems(ctx context.Context, responseID string, limit int, order SortOrder, after string) (ListItemsResponse, error) {
	if ctx.Err() != nil {
		return ListItemsResponse{}, ctx.Err()
	}

	s.mu.Lock()
	state, ok := s.responses[responseID]
	s.mu.Unlock()
	if !ok {
		return ListItemsResponse{}, errors.New("response not found")
	}

	items := append([]ItemResource(nil), state.inputItems...)
	if order == SortOrderDesc {
		for i, j := 0, len(items)-1; i < j; i, j = i+1, j-1 {
			items[i], items[j] = items[j], items[i]
		}
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

	if limit <= 0 {
		limit = 20
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

func (s *InMemoryResponsesService) newResponseID() string {
	return "resp_" + uuid.NewString()
}

func (s *InMemoryResponsesService) newItemID() string {
	return "item_" + uuid.NewString()
}

func (s *InMemoryResponsesService) initResponse(responseID string, request CreateResponse) *responseState {
	model := request.Model
	if model == "" {
		model = s.agent.Name()
	}

	resp := Response{
		ID:                 responseID,
		Object:             "response",
		CreatedAt:          time.Now().Unix(),
		Model:              model,
		Status:             ResponseStatusInProgress,
		Instructions:       request.Instructions,
		PreviousResponseID: request.PreviousResponseID,
		Conversation:       request.Conversation,
		Background:         request.Background,
		Metadata:           request.Metadata,
		Output:             []ItemResource{},
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := &responseState{response: resp}
	state.cond = sync.NewCond(&s.mu)
	s.responses[responseID] = state

	s.appendEventLocked(state, StreamingResponseEvent{
		Type:           "response.created",
		SequenceNumber: s.nextSequence(state),
		Response:       &state.response,
	})
	s.appendEventLocked(state, StreamingResponseEvent{
		Type:           "response.in_progress",
		SequenceNumber: s.nextSequence(state),
		Response:       &state.response,
	})

	return state
}

func (s *InMemoryResponsesService) parseInput(request CreateResponse) ([]agent.Message, []ItemResource, error) {
	if len(request.Input) == 0 {
		return nil, nil, errors.New("input is required")
	}

	var inputText string
	if err := json.Unmarshal(request.Input, &inputText); err == nil {
		msg := agent.NewUserMessage(inputText)
		item := ItemResource{
			ID:        s.newItemID(),
			Type:      "message",
			Role:      "user",
			Content:   inputText,
			CreatedAt: time.Now().Unix(),
		}
		return []agent.Message{msg}, []ItemResource{item}, nil
	}

	var messages []ResponseInputMessage
	if err := json.Unmarshal(request.Input, &messages); err != nil {
		return nil, nil, errors.New("input must be a string or array of messages")
	}

	result := make([]agent.Message, 0, len(messages))
	items := make([]ItemResource, 0, len(messages))
	for _, msg := range messages {
		text, ok := msg.Content.(string)
		if !ok {
			return nil, nil, errors.New("message content must be a string")
		}
		switch msg.Role {
		case "system":
			result = append(result, agent.NewSystemMessage(text))
		case "assistant":
			result = append(result, agent.NewAssistantMessage(text))
		default:
			result = append(result, agent.NewUserMessage(text))
		}

		items = append(items, ItemResource{
			ID:        s.newItemID(),
			Type:      "message",
			Role:      msg.Role,
			Content:   text,
			CreatedAt: time.Now().Unix(),
		})
	}
	return result, items, nil
}

func (s *InMemoryResponsesService) runAgent(ctx context.Context, state *responseState, request CreateResponse, messages []agent.Message) (Response, error) {
	if request.Instructions != "" {
		messages = append([]agent.Message{agent.NewSystemMessage(request.Instructions)}, messages...)
	}

	resp, err := s.agent.Run(ctx, messages)
	if err != nil {
		state.response.Status = ResponseStatusFailed
		state.response.Error = &ResponseError{Message: err.Error()}
		s.mu.Lock()
		s.appendEventLocked(state, StreamingResponseEvent{
			Type:           "response.failed",
			SequenceNumber: s.nextSequence(state),
			Response:       &state.response,
		})
		state.cond.Broadcast()
		s.mu.Unlock()
		return state.response, err
	}

	items := make([]ItemResource, 0, len(resp.Messages))
	for _, msg := range resp.Messages {
		text := msg.Text()
		if text == "" {
			continue
		}
		items = append(items, ItemResource{
			ID:        s.newItemID(),
			Type:      "message",
			Role:      string(msg.Role),
			Content:   text,
			CreatedAt: time.Now().Unix(),
		})
	}

	state.response.Output = items
	state.response.Status = ResponseStatusCompleted
	if resp.Usage != nil {
		state.response.Usage = &ResponseUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	s.mu.Lock()
	for i, item := range items {
		s.appendEventLocked(state, StreamingResponseEvent{
			Type:           "response.output_item.added",
			SequenceNumber: s.nextSequence(state),
			OutputIndex:    i,
			Item:           &item,
		})
		s.appendEventLocked(state, StreamingResponseEvent{
			Type:           "response.output_item.done",
			SequenceNumber: s.nextSequence(state),
			OutputIndex:    i,
			Item:           &item,
		})
	}
	if len(items) == 0 {
		s.appendEventLocked(state, StreamingResponseEvent{
			Type:           "response.output_item.done",
			SequenceNumber: s.nextSequence(state),
			OutputIndex:    0,
		})
	}
	s.appendEventLocked(state, StreamingResponseEvent{
		Type:           "response.completed",
		SequenceNumber: s.nextSequence(state),
		Response:       &state.response,
	})
	state.cond.Broadcast()
	s.mu.Unlock()

	s.addConversationItems(ctx, state)

	return state.response, nil
}

func (s *InMemoryResponsesService) streamAgent(ctx context.Context, ch chan<- StreamingResponseEvent, state *responseState, request CreateResponse, messages []agent.Message) {
	if request.Instructions != "" {
		messages = append([]agent.Message{agent.NewSystemMessage(request.Instructions)}, messages...)
	}

	stream, err := s.agent.RunStream(ctx, messages)
	if err != nil {
		state.response.Status = ResponseStatusFailed
		state.response.Error = &ResponseError{Message: err.Error()}
		s.emitEvent(state, ch, "response.failed", &state.response, 0, nil, "")
		state.done = true
		state.cond.Broadcast()
		return
	}

	item := ItemResource{
		ID:        s.newItemID(),
		Type:      "message",
		Role:      string(chat.RoleAssistant),
		Content:   "",
		CreatedAt: time.Now().Unix(),
	}

	s.emitEvent(state, ch, "response.output_item.added", nil, 0, &item, "")

	text := ""
	for update := range stream {
		if update.Error != nil {
			state.response.Status = ResponseStatusFailed
			state.response.Error = &ResponseError{Message: update.Error.Error()}
			s.emitEvent(state, ch, "response.failed", &state.response, 0, nil, "")
			state.done = true
			state.cond.Broadcast()
			return
		}

		switch update.Kind {
		case agent.UpdateKindContentDelta:
			if update.Delta != nil && update.Delta.TextDelta != "" {
				text += update.Delta.TextDelta
				s.emitEvent(state, ch, "response.output_text.delta", nil, 0, nil, update.Delta.TextDelta)
			}
		case agent.UpdateKindDone:
			item.Content = text
			state.response.Output = []ItemResource{item}
			state.response.Status = ResponseStatusCompleted
			if update.Usage != nil {
				state.response.Usage = &ResponseUsage{
					InputTokens:  update.Usage.InputTokens,
					OutputTokens: update.Usage.OutputTokens,
					TotalTokens:  update.Usage.TotalTokens,
				}
			}
			break
		case agent.UpdateKindUsage:
			if update.Usage != nil {
				state.response.Usage = &ResponseUsage{
					InputTokens:  update.Usage.InputTokens,
					OutputTokens: update.Usage.OutputTokens,
					TotalTokens:  update.Usage.TotalTokens,
				}
			}
		}
	}

	item.Content = text
	state.response.Output = []ItemResource{item}
	state.response.Status = ResponseStatusCompleted

	s.emitEvent(state, ch, "response.output_text.done", nil, 0, nil, text)
	s.emitEvent(state, ch, "response.output_item.done", nil, 0, &item, "")
	s.emitEvent(state, ch, "response.completed", &state.response, 0, nil, "")

	state.done = true
	state.cond.Broadcast()

	s.addConversationItems(ctx, state)
}

func (s *InMemoryResponsesService) addConversationItems(ctx context.Context, state *responseState) {
	if s.conversationStore == nil || state.response.Conversation == nil || state.response.Conversation.ID == "" {
		return
	}

	conversationID := state.response.Conversation.ID
	if _, ok, _ := s.conversationStore.GetConversation(ctx, conversationID); !ok {
		_, _ = s.conversationStore.CreateConversation(ctx, Conversation{
			ID:       conversationID,
			Metadata: state.response.Conversation.Metadata,
		})
	}

	items := append([]ItemResource(nil), state.inputItems...)
	items = append(items, state.response.Output...)
	if len(items) > 0 {
		_ = s.conversationStore.AddItems(ctx, conversationID, items)
	}
}

func (s *InMemoryResponsesService) emitEvent(state *responseState, ch chan<- StreamingResponseEvent, eventType string, response *Response, outputIndex int, item *ItemResource, delta string) {
	s.mu.Lock()
	sequence := s.nextSequence(state)
	event := StreamingResponseEvent{
		Type:           eventType,
		SequenceNumber: sequence,
		Response:       response,
		OutputIndex:    outputIndex,
		Item:           item,
		Delta:          delta,
		Text:           delta,
	}
	s.appendEventLocked(state, event)
	state.cond.Broadcast()
	s.mu.Unlock()

	select {
	case ch <- event:
	default:
		ch <- event
	}
}

func (s *InMemoryResponsesService) nextSequence(state *responseState) int {
	return len(state.events) + 1
}

func (s *InMemoryResponsesService) appendEventLocked(state *responseState, event StreamingResponseEvent) {
	state.events = append(state.events, event)
}
