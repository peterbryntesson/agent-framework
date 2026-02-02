// Copyright (c) Microsoft. All rights reserved.

package chat

import (
	"encoding/json"
	"testing"
	"time"
)

func TestContentType_Constants(t *testing.T) {
	// Arrange & Assert
	if ContentTypeText != "text" {
		t.Errorf("expected ContentTypeText 'text', got '%s'", ContentTypeText)
	}
	if ContentTypeImage != "image" {
		t.Errorf("expected ContentTypeImage 'image', got '%s'", ContentTypeImage)
	}
	if ContentTypeToolCall != "tool_call" {
		t.Errorf("expected ContentTypeToolCall 'tool_call', got '%s'", ContentTypeToolCall)
	}
	if ContentTypeToolResult != "tool_result" {
		t.Errorf("expected ContentTypeToolResult 'tool_result', got '%s'", ContentTypeToolResult)
	}
}

func TestTextContent_Type(t *testing.T) {
	// Arrange
	tc := &TextContent{Text: "Hello"}

	// Act
	contentType := tc.Type()

	// Assert
	if contentType != ContentTypeText {
		t.Errorf("expected ContentTypeText, got '%s'", contentType)
	}
}

func TestTextContent_Fields(t *testing.T) {
	// Arrange
	tc := &TextContent{Text: "Hello, world!"}

	// Act & Assert
	if tc.Text != "Hello, world!" {
		t.Errorf("expected Text 'Hello, world!', got '%s'", tc.Text)
	}
}

func TestNewTextContent(t *testing.T) {
	// Arrange & Act
	tc := NewTextContent("Test message")

	// Assert
	if tc == nil {
		t.Fatal("NewTextContent returned nil")
	}
	if tc.Text != "Test message" {
		t.Errorf("expected Text 'Test message', got '%s'", tc.Text)
	}
	if tc.Type() != ContentTypeText {
		t.Errorf("expected ContentTypeText, got '%s'", tc.Type())
	}
}

func TestImageContent_Type(t *testing.T) {
	// Arrange
	ic := &ImageContent{URL: "https://example.com/image.png"}

	// Act
	contentType := ic.Type()

	// Assert
	if contentType != ContentTypeImage {
		t.Errorf("expected ContentTypeImage, got '%s'", contentType)
	}
}

func TestImageContent_Fields(t *testing.T) {
	// Arrange
	ic := &ImageContent{
		URL:       "https://example.com/image.png",
		MediaType: "image/png",
		Detail:    "high",
	}

	// Act & Assert
	if ic.URL != "https://example.com/image.png" {
		t.Errorf("expected URL 'https://example.com/image.png', got '%s'", ic.URL)
	}
	if ic.MediaType != "image/png" {
		t.Errorf("expected MediaType 'image/png', got '%s'", ic.MediaType)
	}
	if ic.Detail != "high" {
		t.Errorf("expected Detail 'high', got '%s'", ic.Detail)
	}
}

func TestNewImageContentFromURL(t *testing.T) {
	// Arrange & Act
	ic := NewImageContentFromURL("https://example.com/photo.jpg")

	// Assert
	if ic == nil {
		t.Fatal("NewImageContentFromURL returned nil")
	}
	if ic.URL != "https://example.com/photo.jpg" {
		t.Errorf("expected URL 'https://example.com/photo.jpg', got '%s'", ic.URL)
	}
	if ic.Type() != ContentTypeImage {
		t.Errorf("expected ContentTypeImage, got '%s'", ic.Type())
	}
}

func TestNewImageContentFromBase64(t *testing.T) {
	// Arrange
	base64Data := "iVBORw0KGgoAAAANSUhEUg..."
	mediaType := "image/png"

	// Act
	ic := NewImageContentFromBase64(base64Data, mediaType)

	// Assert
	if ic == nil {
		t.Fatal("NewImageContentFromBase64 returned nil")
	}
	if ic.Base64Data != base64Data {
		t.Errorf("expected Base64Data '%s', got '%s'", base64Data, ic.Base64Data)
	}
	if ic.MediaType != mediaType {
		t.Errorf("expected MediaType '%s', got '%s'", mediaType, ic.MediaType)
	}
	if ic.Type() != ContentTypeImage {
		t.Errorf("expected ContentTypeImage, got '%s'", ic.Type())
	}
}

func TestToolCallContent_Type(t *testing.T) {
	// Arrange
	tcc := &ToolCallContent{
		ToolCall: ToolCall{ID: "call_123", Name: "get_weather"},
	}

	// Act
	contentType := tcc.Type()

	// Assert
	if contentType != ContentTypeToolCall {
		t.Errorf("expected ContentTypeToolCall, got '%s'", contentType)
	}
}

func TestToolCall_Fields(t *testing.T) {
	// Arrange
	args := json.RawMessage(`{"location": "Seattle"}`)
	tc := ToolCall{
		ID:        "call_abc123",
		Name:      "get_weather",
		Arguments: args,
	}

	// Act & Assert
	if tc.ID != "call_abc123" {
		t.Errorf("expected ID 'call_abc123', got '%s'", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("expected Name 'get_weather', got '%s'", tc.Name)
	}
	if string(tc.Arguments) != `{"location": "Seattle"}` {
		t.Errorf("expected Arguments '{\"location\": \"Seattle\"}', got '%s'", string(tc.Arguments))
	}
}

func TestNewToolCallContent(t *testing.T) {
	// Arrange
	args := json.RawMessage(`{"city": "Paris"}`)

	// Act
	tcc := NewToolCallContent("call_xyz", "search", args)

	// Assert
	if tcc == nil {
		t.Fatal("NewToolCallContent returned nil")
	}
	if tcc.ToolCall.ID != "call_xyz" {
		t.Errorf("expected ID 'call_xyz', got '%s'", tcc.ToolCall.ID)
	}
	if tcc.ToolCall.Name != "search" {
		t.Errorf("expected Name 'search', got '%s'", tcc.ToolCall.Name)
	}
	if string(tcc.ToolCall.Arguments) != `{"city": "Paris"}` {
		t.Errorf("expected Arguments '{\"city\": \"Paris\"}', got '%s'", string(tcc.ToolCall.Arguments))
	}
	if tcc.Type() != ContentTypeToolCall {
		t.Errorf("expected ContentTypeToolCall, got '%s'", tcc.Type())
	}
}

func TestToolResultContent_Type(t *testing.T) {
	// Arrange
	trc := &ToolResultContent{ToolCallID: "call_123", Content: "result"}

	// Act
	contentType := trc.Type()

	// Assert
	if contentType != ContentTypeToolResult {
		t.Errorf("expected ContentTypeToolResult, got '%s'", contentType)
	}
}

func TestToolResultContent_Fields(t *testing.T) {
	// Arrange
	trc := &ToolResultContent{
		ToolCallID: "call_abc",
		Content:    `{"temperature": 72}`,
		IsError:    false,
	}

	// Act & Assert
	if trc.ToolCallID != "call_abc" {
		t.Errorf("expected ToolCallID 'call_abc', got '%s'", trc.ToolCallID)
	}
	if trc.Content != `{"temperature": 72}` {
		t.Errorf("expected Content '{\"temperature\": 72}', got '%s'", trc.Content)
	}
	if trc.IsError {
		t.Error("expected IsError false, got true")
	}
}

func TestNewToolResultContent(t *testing.T) {
	// Arrange & Act
	trc := NewToolResultContent("call_result_123", "Success!")

	// Assert
	if trc == nil {
		t.Fatal("NewToolResultContent returned nil")
	}
	if trc.ToolCallID != "call_result_123" {
		t.Errorf("expected ToolCallID 'call_result_123', got '%s'", trc.ToolCallID)
	}
	if trc.Content != "Success!" {
		t.Errorf("expected Content 'Success!', got '%s'", trc.Content)
	}
	if trc.IsError {
		t.Error("expected IsError false, got true")
	}
	if trc.Type() != ContentTypeToolResult {
		t.Errorf("expected ContentTypeToolResult, got '%s'", trc.Type())
	}
}

func TestNewToolResultContentWithError(t *testing.T) {
	// Arrange & Act
	trc := NewToolResultContentWithError("call_err", "Something went wrong")

	// Assert
	if trc == nil {
		t.Fatal("NewToolResultContentWithError returned nil")
	}
	if trc.ToolCallID != "call_err" {
		t.Errorf("expected ToolCallID 'call_err', got '%s'", trc.ToolCallID)
	}
	if trc.Content != "Something went wrong" {
		t.Errorf("expected Content 'Something went wrong', got '%s'", trc.Content)
	}
	if !trc.IsError {
		t.Error("expected IsError true, got false")
	}
}

func TestContent_Implementations(t *testing.T) {
	// Verify all content types implement the Content interface
	var _ Content = (*TextContent)(nil)
	var _ Content = (*ImageContent)(nil)
	var _ Content = (*ToolCallContent)(nil)
	var _ Content = (*ToolResultContent)(nil)
}

func TestContent_TypesAreDistinct(t *testing.T) {
	// Arrange
	types := []ContentType{
		ContentTypeText,
		ContentTypeImage,
		ContentTypeToolCall,
		ContentTypeToolResult,
	}

	// Act & Assert
	seen := make(map[ContentType]bool)
	for _, ct := range types {
		if seen[ct] {
			t.Errorf("duplicate ContentType: %s", ct)
		}
		seen[ct] = true
	}
}

func TestMessage_Text_NilMessage(t *testing.T) {
	// Arrange
	var msg *Message = nil

	// Act
	text := msg.Text()

	// Assert
	if text != "" {
		t.Errorf("expected empty string for nil message, got '%s'", text)
	}
}

func TestMessage_Text_EmptyContents(t *testing.T) {
	// Arrange
	msg := &Message{Role: RoleUser, Contents: []Content{}}

	// Act
	text := msg.Text()

	// Assert
	if text != "" {
		t.Errorf("expected empty string for empty contents, got '%s'", text)
	}
}

func TestMessage_Text_SingleTextContent(t *testing.T) {
	// Arrange
	msg := &Message{
		Role:     RoleUser,
		Contents: []Content{NewTextContent("Hello")},
	}

	// Act
	text := msg.Text()

	// Assert
	if text != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", text)
	}
}

func TestMessage_Text_MultipleTextContents(t *testing.T) {
	// Arrange
	msg := &Message{
		Role: RoleAssistant,
		Contents: []Content{
			NewTextContent("Hello, "),
			NewTextContent("world!"),
		},
	}

	// Act
	text := msg.Text()

	// Assert
	if text != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got '%s'", text)
	}
}

func TestMessage_Text_MixedContents(t *testing.T) {
	// Arrange
	msg := &Message{
		Role: RoleUser,
		Contents: []Content{
			NewTextContent("Look at this: "),
			NewImageContentFromURL("https://example.com/image.png"),
			NewTextContent(" What do you think?"),
		},
	}

	// Act
	text := msg.Text()

	// Assert
	expected := "Look at this:  What do you think?"
	if text != expected {
		t.Errorf("expected '%s', got '%s'", expected, text)
	}
}

func TestMessage_Text_NoTextContent(t *testing.T) {
	// Arrange
	msg := &Message{
		Role: RoleUser,
		Contents: []Content{
			NewImageContentFromURL("https://example.com/image.png"),
		},
	}

	// Act
	text := msg.Text()

	// Assert
	if text != "" {
		t.Errorf("expected empty string when no text content, got '%s'", text)
	}
}

func TestNewToolMessage(t *testing.T) {
	// Arrange
	before := time.Now()

	// Act
	msg := NewToolMessage("call_123", "Tool result data")

	// Assert
	after := time.Now()
	if msg.Role != RoleTool {
		t.Errorf("expected Role 'tool', got '%s'", msg.Role)
	}
	if msg.ToolCallID != "call_123" {
		t.Errorf("expected ToolCallID 'call_123', got '%s'", msg.ToolCallID)
	}
	if msg.Text() != "Tool result data" {
		t.Errorf("expected Text() 'Tool result data', got '%s'", msg.Text())
	}
	if msg.CreatedAt.Before(before) || msg.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after time")
	}
}

func TestNewAssistantMessageWithToolCalls(t *testing.T) {
	// Arrange
	before := time.Now()
	toolCalls := []ToolCall{
		{ID: "call_1", Name: "get_weather", Arguments: json.RawMessage(`{"city":"NYC"}`)},
		{ID: "call_2", Name: "get_time", Arguments: json.RawMessage(`{"zone":"EST"}`)},
	}

	// Act
	msg := NewAssistantMessageWithToolCalls(toolCalls)

	// Assert
	after := time.Now()
	if msg.Role != RoleAssistant {
		t.Errorf("expected Role 'assistant', got '%s'", msg.Role)
	}
	if len(msg.ToolCalls) != 2 {
		t.Errorf("expected 2 tool calls, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID != "call_1" {
		t.Errorf("expected first ToolCall ID 'call_1', got '%s'", msg.ToolCalls[0].ID)
	}
	if msg.ToolCalls[1].Name != "get_time" {
		t.Errorf("expected second ToolCall Name 'get_time', got '%s'", msg.ToolCalls[1].Name)
	}
	if msg.CreatedAt.Before(before) || msg.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after time")
	}
}

func TestNewMessageWithContents(t *testing.T) {
	// Arrange
	before := time.Now()
	contents := []Content{
		NewTextContent("Check this image: "),
		NewImageContentFromURL("https://example.com/photo.jpg"),
	}

	// Act
	msg := NewMessageWithContents(RoleUser, contents...)

	// Assert
	after := time.Now()
	if msg.Role != RoleUser {
		t.Errorf("expected Role 'user', got '%s'", msg.Role)
	}
	if len(msg.Contents) != 2 {
		t.Errorf("expected 2 content items, got %d", len(msg.Contents))
	}
	if msg.Text() != "Check this image: " {
		t.Errorf("expected Text() 'Check this image: ', got '%s'", msg.Text())
	}
	if msg.CreatedAt.Before(before) || msg.CreatedAt.After(after) {
		t.Error("CreatedAt should be between before and after time")
	}
}

func TestMessage_ToolCalls_Field(t *testing.T) {
	// Arrange
	msg := Message{
		Role: RoleAssistant,
		ToolCalls: []ToolCall{
			{ID: "call_abc", Name: "search", Arguments: json.RawMessage(`{"q":"test"}`)},
		},
	}

	// Act & Assert
	if len(msg.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID != "call_abc" {
		t.Errorf("expected ID 'call_abc', got '%s'", msg.ToolCalls[0].ID)
	}
}

func TestTextContent_JSON(t *testing.T) {
	// Arrange
	tc := NewTextContent("Hello, world!")

	// Act
	data, err := json.Marshal(tc)

	// Assert
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	expected := `{"text":"Hello, world!"}`
	if string(data) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(data))
	}
}

func TestImageContent_JSON(t *testing.T) {
	// Arrange
	ic := &ImageContent{
		URL:       "https://example.com/img.png",
		MediaType: "image/png",
		Detail:    "low",
	}

	// Act
	data, err := json.Marshal(ic)

	// Assert
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	// Verify it contains expected fields
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["url"] != "https://example.com/img.png" {
		t.Errorf("expected url 'https://example.com/img.png', got '%v'", result["url"])
	}
	if result["media_type"] != "image/png" {
		t.Errorf("expected media_type 'image/png', got '%v'", result["media_type"])
	}
}

func TestToolCallContent_JSON(t *testing.T) {
	// Arrange
	tcc := NewToolCallContent("call_123", "get_data", json.RawMessage(`{"id":42}`))

	// Act
	data, err := json.Marshal(tcc)

	// Assert
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	toolCall := result["tool_call"].(map[string]interface{})
	if toolCall["id"] != "call_123" {
		t.Errorf("expected id 'call_123', got '%v'", toolCall["id"])
	}
	if toolCall["name"] != "get_data" {
		t.Errorf("expected name 'get_data', got '%v'", toolCall["name"])
	}
}

func TestToolResultContent_JSON(t *testing.T) {
	// Arrange
	trc := NewToolResultContentWithError("call_err", "Error occurred")

	// Act
	data, err := json.Marshal(trc)

	// Assert
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
	if result["tool_call_id"] != "call_err" {
		t.Errorf("expected tool_call_id 'call_err', got '%v'", result["tool_call_id"])
	}
	if result["is_error"] != true {
		t.Errorf("expected is_error true, got '%v'", result["is_error"])
	}
}
