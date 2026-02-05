// Copyright (c) Microsoft. All rights reserved.

package chat

import (
	"context"
	"testing"
	"time"
)

// mockClient is a test implementation of the Client interface.
type mockClient struct {
	metadata         ClientMetadata
	getResponseFunc  func(ctx context.Context, messages []Message, options *Options) (*Response, error)
	getStreamingFunc func(ctx context.Context, messages []Message, options *Options) (<-chan ResponseUpdate, error)
}

func (m *mockClient) GetResponse(ctx context.Context, messages []Message, options *Options) (*Response, error) {
	if m.getResponseFunc != nil {
		return m.getResponseFunc(ctx, messages, options)
	}
	return &Response{
		Message: Message{
			Role:     RoleAssistant,
			Contents: []Content{NewTextContent("Hello!")},
		},
		FinishReason: FinishReasonStop,
	}, nil
}

func (m *mockClient) GetStreamingResponse(ctx context.Context, messages []Message, options *Options) (<-chan ResponseUpdate, error) {
	if m.getStreamingFunc != nil {
		return m.getStreamingFunc(ctx, messages, options)
	}
	ch := make(chan ResponseUpdate, 1)
	ch <- ResponseUpdate{Kind: UpdateKindDone}
	close(ch)
	return ch, nil
}

func (m *mockClient) Metadata() ClientMetadata {
	return m.metadata
}

// Verify mockClient implements Client interface.
var _ Client = (*mockClient)(nil)

func TestClientMetadata_Fields(t *testing.T) {
	// Arrange
	meta := ClientMetadata{
		ProviderName: "openai",
		ModelID:      "gpt-4",
		EndpointURI:  "https://api.openai.com/v1",
	}

	// Act & Assert
	if meta.ProviderName != "openai" {
		t.Errorf("expected ProviderName 'openai', got '%s'", meta.ProviderName)
	}
	if meta.ModelID != "gpt-4" {
		t.Errorf("expected ModelID 'gpt-4', got '%s'", meta.ModelID)
	}
	if meta.EndpointURI != "https://api.openai.com/v1" {
		t.Errorf("expected EndpointURI 'https://api.openai.com/v1', got '%s'", meta.EndpointURI)
	}
}

func TestClientInterface_GetResponse(t *testing.T) {
	// Arrange
	client := &mockClient{
		metadata: ClientMetadata{ProviderName: "test"},
		getResponseFunc: func(_ context.Context, messages []Message, _ *Options) (*Response, error) {
			return &Response{
				Message: Message{
					Role:     RoleAssistant,
					Contents: []Content{NewTextContent("Response to: " + messages[0].Text())},
				},
				FinishReason: FinishReasonStop,
			}, nil
		},
	}
	messages := []Message{NewUserMessage("Hello")}

	// Act
	response, err := client.GetResponse(context.Background(), messages, nil)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if response.Text() != "Response to: Hello" {
		t.Errorf("expected 'Response to: Hello', got '%s'", response.Text())
	}
}

func TestClientInterface_GetStreamingResponse(t *testing.T) {
	// Arrange
	client := &mockClient{
		metadata: ClientMetadata{ProviderName: "test"},
		getStreamingFunc: func(_ context.Context, _ []Message, _ *Options) (<-chan ResponseUpdate, error) {
			ch := make(chan ResponseUpdate, 3)
			ch <- ResponseUpdate{
				Kind:  UpdateKindContentDelta,
				Delta: &ContentDelta{TextDelta: "Hello"},
			}
			ch <- ResponseUpdate{
				Kind:  UpdateKindContentDelta,
				Delta: &ContentDelta{TextDelta: " World"},
			}
			ch <- ResponseUpdate{Kind: UpdateKindDone}
			close(ch)
			return ch, nil
		},
	}
	messages := []Message{NewUserMessage("Hi")}

	// Act
	updates, err := client.GetStreamingResponse(context.Background(), messages, nil)

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var content string
	for update := range updates {
		if update.Kind == UpdateKindContentDelta {
			content += update.Delta.TextDelta
		}
	}
	if content != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", content)
	}
}

func TestClientInterface_Metadata(t *testing.T) {
	// Arrange
	expectedMeta := ClientMetadata{
		ProviderName: "azure",
		ModelID:      "gpt-4o",
		EndpointURI:  "https://myaccount.openai.azure.com",
	}
	client := &mockClient{metadata: expectedMeta}

	// Act
	meta := client.Metadata()

	// Assert
	if meta.ProviderName != expectedMeta.ProviderName {
		t.Errorf("expected ProviderName '%s', got '%s'", expectedMeta.ProviderName, meta.ProviderName)
	}
	if meta.ModelID != expectedMeta.ModelID {
		t.Errorf("expected ModelID '%s', got '%s'", expectedMeta.ModelID, meta.ModelID)
	}
	if meta.EndpointURI != expectedMeta.EndpointURI {
		t.Errorf("expected EndpointURI '%s', got '%s'", expectedMeta.EndpointURI, meta.EndpointURI)
	}
}

func TestOptions_NewOptions(t *testing.T) {
	// Arrange & Act
	opts := NewOptions()

	// Assert
	if opts == nil {
		t.Fatal("NewOptions returned nil")
	}
	if opts.Metadata == nil {
		t.Error("Metadata map should be initialized")
	}
	if opts.MaxTokens != 0 {
		t.Errorf("expected MaxTokens 0, got %d", opts.MaxTokens)
	}
	if opts.Temperature != 0 {
		t.Errorf("expected Temperature 0, got %f", opts.Temperature)
	}
}

func TestOptions_Fields(t *testing.T) {
	// Arrange
	opts := &Options{
		MaxTokens:      1000,
		Temperature:    0.7,
		TopP:           0.9,
		StopSequences:  []string{"END"},
		ResponseFormat: "json_object",
		Metadata:       map[string]interface{}{"key": "value"},
	}

	// Act & Assert
	if opts.MaxTokens != 1000 {
		t.Errorf("expected MaxTokens 1000, got %d", opts.MaxTokens)
	}
	if opts.Temperature != 0.7 {
		t.Errorf("expected Temperature 0.7, got %f", opts.Temperature)
	}
	if opts.TopP != 0.9 {
		t.Errorf("expected TopP 0.9, got %f", opts.TopP)
	}
	if len(opts.StopSequences) != 1 || opts.StopSequences[0] != "END" {
		t.Errorf("unexpected StopSequences: %v", opts.StopSequences)
	}
	if opts.ResponseFormat != "json_object" {
		t.Errorf("expected ResponseFormat 'json_object', got '%s'", opts.ResponseFormat)
	}
	if opts.Metadata["key"] != "value" {
		t.Errorf("expected Metadata['key'] = 'value', got '%v'", opts.Metadata["key"])
	}
}

func TestResponseText_NilResponse(t *testing.T) {
	// Arrange
	var response *Response

	// Act
	text := response.Text()

	// Assert
	if text != "" {
		t.Errorf("expected empty string for nil response, got '%s'", text)
	}
}

func TestResponseText_EmptyContent(t *testing.T) {
	// Arrange
	response := &Response{
		Message: Message{Role: RoleAssistant, Contents: []Content{}},
	}

	// Act
	text := response.Text()

	// Assert
	if text != "" {
		t.Errorf("expected empty string, got '%s'", text)
	}
}

func TestResponseText_WithContent(t *testing.T) {
	// Arrange
	response := &Response{
		Message: Message{Role: RoleAssistant, Contents: []Content{NewTextContent("Hello, world!")}},
	}

	// Act
	text := response.Text()

	// Assert
	if text != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got '%s'", text)
	}
}

func TestResponse_AllFields(t *testing.T) {
	// Arrange
	raw := map[string]interface{}{"id": "chatcmpl-123"}
	response := &Response{
		Message: Message{
			Role:     RoleAssistant,
			Contents: []Content{NewTextContent("Test response")},
		},
		FinishReason: FinishReasonStop,
		Usage: &UsageDetails{
			InputTokens:  10,
			OutputTokens: 5,
			TotalTokens:  15,
		},
		RawRepresentation: raw,
	}

	// Act & Assert
	if response.Message.Role != RoleAssistant {
		t.Errorf("expected Role 'assistant', got '%s'", response.Message.Role)
	}
	if response.FinishReason != FinishReasonStop {
		t.Errorf("expected FinishReasonStop, got %d", response.FinishReason)
	}
	if response.Usage.InputTokens != 10 {
		t.Errorf("expected InputTokens 10, got %d", response.Usage.InputTokens)
	}
	if response.RawRepresentation == nil {
		t.Error("expected RawRepresentation to be set")
	}
}

func TestUpdateKind_Constants(t *testing.T) {
	// Verify enum values are distinct and in expected order
	kinds := []UpdateKind{
		UpdateKindContentDelta,
		UpdateKindToolCall,
		UpdateKindToolResult,
		UpdateKindMessageComplete,
		UpdateKindUsage,
		UpdateKindError,
		UpdateKindDone,
	}

	seen := make(map[UpdateKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate UpdateKind value: %d", k)
		}
		seen[k] = true
	}
}

func TestFinishReason_Constants(t *testing.T) {
	// Verify enum values are distinct
	reasons := []FinishReason{
		FinishReasonStop,
		FinishReasonLength,
		FinishReasonToolCalls,
		FinishReasonContentFilter,
	}

	seen := make(map[FinishReason]bool)
	for _, r := range reasons {
		if seen[r] {
			t.Errorf("duplicate FinishReason value: %d", r)
		}
		seen[r] = true
	}
}

func TestRole_Constants(t *testing.T) {
	// Verify role string values
	if RoleSystem != "system" {
		t.Errorf("expected RoleSystem 'system', got '%s'", RoleSystem)
	}
	if RoleUser != "user" {
		t.Errorf("expected RoleUser 'user', got '%s'", RoleUser)
	}
	if RoleAssistant != "assistant" {
		t.Errorf("expected RoleAssistant 'assistant', got '%s'", RoleAssistant)
	}
	if RoleTool != "tool" {
		t.Errorf("expected RoleTool 'tool', got '%s'", RoleTool)
	}
}

func TestNewUserMessage(t *testing.T) {
	// Arrange
	before := time.Now()

	// Act
	msg := NewUserMessage("Hello")

	// Assert
	after := time.Now()
	if msg.Role != RoleUser {
		t.Errorf("expected Role 'user', got '%s'", msg.Role)
	}
	if msg.Text() != "Hello" {
		t.Errorf("expected Text() 'Hello', got '%s'", msg.Text())
	}
	if len(msg.Contents) != 1 {
		t.Errorf("expected 1 content item, got %d", len(msg.Contents))
	}
	if msg.CreatedAt.Before(before) || msg.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after time")
	}
}

func TestNewSystemMessage(t *testing.T) {
	// Arrange
	before := time.Now()

	// Act
	msg := NewSystemMessage("You are helpful")

	// Assert
	after := time.Now()
	if msg.Role != RoleSystem {
		t.Errorf("expected Role 'system', got '%s'", msg.Role)
	}
	if msg.Text() != "You are helpful" {
		t.Errorf("expected Text() 'You are helpful', got '%s'", msg.Text())
	}
	if len(msg.Contents) != 1 {
		t.Errorf("expected 1 content item, got %d", len(msg.Contents))
	}
	if msg.CreatedAt.Before(before) || msg.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after time")
	}
}

func TestNewAssistantMessage(t *testing.T) {
	// Arrange
	before := time.Now()

	// Act
	msg := NewAssistantMessage("I can help")

	// Assert
	after := time.Now()
	if msg.Role != RoleAssistant {
		t.Errorf("expected Role 'assistant', got '%s'", msg.Role)
	}
	if msg.Text() != "I can help" {
		t.Errorf("expected Text() 'I can help', got '%s'", msg.Text())
	}
	if len(msg.Contents) != 1 {
		t.Errorf("expected 1 content item, got %d", len(msg.Contents))
	}
	if msg.CreatedAt.Before(before) || msg.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after time")
	}
}

func TestMessage_AllFields(t *testing.T) {
	// Arrange
	now := time.Now()
	raw := map[string]interface{}{"provider": "test"}
	msg := Message{
		Role:              RoleTool,
		Contents:          []Content{NewTextContent("Tool result")},
		Name:              "get_weather",
		ToolCallID:        "call_123",
		CreatedAt:         now,
		RawRepresentation: raw,
	}

	// Act & Assert
	if msg.Role != RoleTool {
		t.Errorf("expected Role 'tool', got '%s'", msg.Role)
	}
	if msg.Text() != "Tool result" {
		t.Errorf("expected Text() 'Tool result', got '%s'", msg.Text())
	}
	if msg.Name != "get_weather" {
		t.Errorf("expected Name 'get_weather', got '%s'", msg.Name)
	}
	if msg.ToolCallID != "call_123" {
		t.Errorf("expected ToolCallID 'call_123', got '%s'", msg.ToolCallID)
	}
	if !msg.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, msg.CreatedAt)
	}
	if msg.RawRepresentation == nil {
		t.Error("expected RawRepresentation to be set")
	}
}

func TestUsageDetails_AllFields(t *testing.T) {
	// Arrange
	usage := &UsageDetails{
		InputTokens:     100,
		OutputTokens:    50,
		TotalTokens:     150,
		CachedTokens:    20,
		ReasoningTokens: 10,
	}

	// Act & Assert
	if usage.InputTokens != 100 {
		t.Errorf("expected InputTokens 100, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("expected OutputTokens 50, got %d", usage.OutputTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("expected TotalTokens 150, got %d", usage.TotalTokens)
	}
	if usage.CachedTokens != 20 {
		t.Errorf("expected CachedTokens 20, got %d", usage.CachedTokens)
	}
	if usage.ReasoningTokens != 10 {
		t.Errorf("expected ReasoningTokens 10, got %d", usage.ReasoningTokens)
	}
}

func TestContentDelta_AllFields(t *testing.T) {
	// Arrange
	delta := &ContentDelta{
		Role:       RoleAssistant,
		TextDelta:  "Hello",
		ToolCallID: "call_456",
		Name:       "my_function",
		ArgsDelta:  `{"arg": "value"}`,
	}

	// Act & Assert
	if delta.Role != RoleAssistant {
		t.Errorf("expected Role 'assistant', got '%s'", delta.Role)
	}
	if delta.TextDelta != "Hello" {
		t.Errorf("expected TextDelta 'Hello', got '%s'", delta.TextDelta)
	}
	if delta.ToolCallID != "call_456" {
		t.Errorf("expected ToolCallID 'call_456', got '%s'", delta.ToolCallID)
	}
	if delta.Name != "my_function" {
		t.Errorf("expected Name 'my_function', got '%s'", delta.Name)
	}
	if delta.ArgsDelta != `{"arg": "value"}` {
		t.Errorf("unexpected ArgsDelta: %s", delta.ArgsDelta)
	}
}

func TestResponseUpdate_AllFields(t *testing.T) {
	// Arrange
	update := ResponseUpdate{
		Kind: UpdateKindContentDelta,
		Delta: &ContentDelta{
			TextDelta: "Test",
		},
		Message: &Message{
			Role:     RoleAssistant,
			Contents: []Content{NewTextContent("Complete message")},
		},
		Usage: &UsageDetails{
			TotalTokens: 100,
		},
		FinishReason: FinishReasonStop,
		Error:        nil,
		Metadata:     map[string]interface{}{"key": "value"},
	}

	// Act & Assert
	if update.Kind != UpdateKindContentDelta {
		t.Errorf("expected UpdateKindContentDelta, got %d", update.Kind)
	}
	if update.Delta.TextDelta != "Test" {
		t.Errorf("expected Delta.TextDelta 'Test', got '%s'", update.Delta.TextDelta)
	}
	if update.Message.Text() != "Complete message" {
		t.Errorf("expected Message.Text() 'Complete message', got '%s'", update.Message.Text())
	}
	if update.Usage.TotalTokens != 100 {
		t.Errorf("expected Usage.TotalTokens 100, got %d", update.Usage.TotalTokens)
	}
	if update.Metadata["key"] != "value" {
		t.Errorf("expected Metadata['key'] = 'value', got '%v'", update.Metadata["key"])
	}
}
