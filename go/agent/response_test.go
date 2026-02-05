// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/chat"
)

func TestResponseText_EmptyMessages(t *testing.T) {
	response := &Response{
		Messages: []Message{},
	}

	result := response.Text()

	if result != "" {
		t.Errorf("Expected empty string, got %q", result)
	}
}

func TestResponseText_NilResponse(t *testing.T) {
	var response *Response

	result := response.Text()

	if result != "" {
		t.Errorf("Expected empty string for nil response, got %q", result)
	}
}

func TestResponseText_SingleMessage(t *testing.T) {
	response := &Response{
		Messages: []Message{
			chat.NewAssistantMessage("Hello, world!"),
		},
	}

	result := response.Text()

	expected := "Hello, world!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestResponseText_MultipleMessages(t *testing.T) {
	response := &Response{
		Messages: []Message{
			chat.NewAssistantMessage("Hello, "),
			chat.NewAssistantMessage("world!"),
		},
	}

	result := response.Text()

	expected := "Hello, world!"
	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestResponseFields(t *testing.T) {
	usage := &UsageDetails{
		InputTokens:  10,
		OutputTokens: 20,
		TotalTokens:  30,
	}
	sessionState := json.RawMessage(`{"key": "value"}`)
	metadata := map[string]interface{}{"meta": "data"}
	additionalProps := map[string]interface{}{"extra": "prop"}

	response := &Response{
		Metadata:             metadata,
		Usage:                usage,
		SessionState:         sessionState,
		Messages:             []Message{chat.NewAssistantMessage("test")},
		FinishReason:         FinishReasonStop,
		ContinuationToken:    "token123",
		AdditionalProperties: additionalProps,
		RawRepresentation:    map[string]string{"raw": "data"},
	}

	if response.Usage.InputTokens != 10 {
		t.Errorf("Expected InputTokens 10, got %d", response.Usage.InputTokens)
	}
	if response.FinishReason != FinishReasonStop {
		t.Errorf("Expected FinishReasonStop, got %d", response.FinishReason)
	}
	if response.ContinuationToken != "token123" {
		t.Errorf("Expected ContinuationToken 'token123', got %q", response.ContinuationToken)
	}
	if response.AdditionalProperties["extra"] != "prop" {
		t.Errorf("Expected AdditionalProperties['extra'] = 'prop', got %v", response.AdditionalProperties["extra"])
	}
	if response.RawRepresentation == nil {
		t.Error("Expected RawRepresentation to be set")
	}
	if response.Metadata["meta"] != "data" {
		t.Errorf("Expected Metadata['meta'] = 'data', got %v", response.Metadata["meta"])
	}
	if string(response.SessionState) != `{"key": "value"}` {
		t.Errorf("Expected SessionState to match, got %s", string(response.SessionState))
	}
	if len(response.Messages) != 1 || response.Messages[0].Text() != "test" {
		t.Errorf("Expected one message with content 'test', got %v", response.Messages)
	}
}

func TestAsyncRunStatusIsTerminal_TerminalStatuses(t *testing.T) {
	terminalStatuses := []AsyncRunStatus{
		StatusCompleted,
		StatusCanceled,
		StatusFailed,
		StatusExpired,
	}

	for _, status := range terminalStatuses {
		if !status.IsTerminal() {
			t.Errorf("Expected %s to be terminal", status)
		}
	}
}

func TestAsyncRunStatusIsTerminal_NonTerminalStatuses(t *testing.T) {
	nonTerminalStatuses := []AsyncRunStatus{
		StatusQueued,
		StatusInProgress,
		StatusRequiresAction,
	}

	for _, status := range nonTerminalStatuses {
		if status.IsTerminal() {
			t.Errorf("Expected %s to be non-terminal", status)
		}
	}
}

func TestAsyncRunStatusIsTerminal_UnknownStatus(t *testing.T) {
	unknownStatus := AsyncRunStatus("unknown")

	if unknownStatus.IsTerminal() {
		t.Error("Expected unknown status to be non-terminal")
	}
}

func TestAsyncRunContent_AllFields(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(time.Hour)
	completedAt := now.Add(time.Minute)

	runContent := &AsyncRunContent{
		RunID:       "run_123",
		Status:      StatusCompleted,
		ThreadID:    "thread_456",
		ExpiresAt:   &expiresAt,
		StartedAt:   &now,
		CompletedAt: &completedAt,
		Error:       nil,
	}

	if runContent.RunID != "run_123" {
		t.Errorf("Expected RunID 'run_123', got %q", runContent.RunID)
	}
	if runContent.Status != StatusCompleted {
		t.Errorf("Expected StatusCompleted, got %s", runContent.Status)
	}
	if runContent.ThreadID != "thread_456" {
		t.Errorf("Expected ThreadID 'thread_456', got %q", runContent.ThreadID)
	}
	if runContent.ExpiresAt == nil || !runContent.ExpiresAt.Equal(expiresAt) {
		t.Errorf("Expected ExpiresAt to match, got %v", runContent.ExpiresAt)
	}
	if runContent.StartedAt == nil || !runContent.StartedAt.Equal(now) {
		t.Errorf("Expected StartedAt to match, got %v", runContent.StartedAt)
	}
	if runContent.CompletedAt == nil || !runContent.CompletedAt.Equal(completedAt) {
		t.Errorf("Expected CompletedAt to match, got %v", runContent.CompletedAt)
	}
	if runContent.Error != nil {
		t.Errorf("Expected Error to be nil, got %v", runContent.Error)
	}
}

func TestAsyncRunContent_WithError(t *testing.T) {
	runError := &AsyncRunError{
		Code:    "rate_limit_exceeded",
		Message: "Too many requests",
	}

	runContent := &AsyncRunContent{
		RunID:  "run_789",
		Status: StatusFailed,
		Error:  runError,
	}

	if runContent.Error == nil {
		t.Error("Expected Error to be set")
	}
	if runContent.Error.Code != "rate_limit_exceeded" {
		t.Errorf("Expected error code 'rate_limit_exceeded', got %q", runContent.Error.Code)
	}
	if runContent.Error.Message != "Too many requests" {
		t.Errorf("Expected error message 'Too many requests', got %q", runContent.Error.Message)
	}
	if runContent.RunID != "run_789" {
		t.Errorf("Expected RunID 'run_789', got %q", runContent.RunID)
	}
	if runContent.Status != StatusFailed {
		t.Errorf("Expected StatusFailed, got %s", runContent.Status)
	}
}

func TestUpdateKindConstants(t *testing.T) {
	// Verify all UpdateKind constants are distinct and in expected order
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
	for i, kind := range kinds {
		if seen[kind] {
			t.Errorf("Duplicate UpdateKind value at index %d: %d", i, kind)
		}
		seen[kind] = true

		if int(kind) != i {
			t.Errorf("Expected UpdateKind at index %d to have value %d, got %d", i, i, kind)
		}
	}
}

func TestFinishReasonConstants(t *testing.T) {
	// Verify all FinishReason constants are distinct
	reasons := []FinishReason{
		FinishReasonStop,
		FinishReasonLength,
		FinishReasonToolCalls,
		FinishReasonContentFilter,
	}

	seen := make(map[FinishReason]bool)
	for i, reason := range reasons {
		if seen[reason] {
			t.Errorf("Duplicate FinishReason value at index %d: %d", i, reason)
		}
		seen[reason] = true
	}
}

func TestResponseUpdate_AllFields(t *testing.T) {
	delta := &ContentDelta{
		Role:      "assistant",
		TextDelta: "Hello",
	}
	msg := chat.NewAssistantMessage("Complete message")
	usage := &UsageDetails{TotalTokens: 100}
	metadata := map[string]interface{}{"key": "value"}
	testErr := errors.New("test error")

	update := &ResponseUpdate{
		Kind:         UpdateKindContentDelta,
		Delta:        delta,
		Message:      &msg,
		Usage:        usage,
		FinishReason: FinishReasonStop,
		Error:        testErr,
		Metadata:     metadata,
	}

	if update.Kind != UpdateKindContentDelta {
		t.Errorf("Expected UpdateKindContentDelta, got %d", update.Kind)
	}
	if update.Delta.TextDelta != "Hello" {
		t.Errorf("Expected Delta.TextDelta 'Hello', got %q", update.Delta.TextDelta)
	}
	if update.Message.Text() != "Complete message" {
		t.Errorf("Expected Message.Text() 'Complete message', got %q", update.Message.Text())
	}
	if update.Usage.TotalTokens != 100 {
		t.Errorf("Expected Usage.TotalTokens 100, got %d", update.Usage.TotalTokens)
	}
	if update.FinishReason != FinishReasonStop {
		t.Errorf("Expected FinishReasonStop, got %d", update.FinishReason)
	}
	if update.Error == nil || update.Error.Error() != "test error" {
		t.Errorf("Expected Error 'test error', got %v", update.Error)
	}
	if update.Metadata["key"] != "value" {
		t.Errorf("Expected Metadata['key'] = 'value', got %v", update.Metadata["key"])
	}
}

func TestContentDelta_AllFields(t *testing.T) {
	delta := &ContentDelta{
		Role:       "assistant",
		TextDelta:  "partial text",
		ToolCallID: "call_123",
		Name:       "get_weather",
		ArgsDelta:  `{"location": "Seattle"}`,
	}

	if delta.Role != "assistant" {
		t.Errorf("Expected Role 'assistant', got %q", delta.Role)
	}
	if delta.TextDelta != "partial text" {
		t.Errorf("Expected TextDelta 'partial text', got %q", delta.TextDelta)
	}
	if delta.ToolCallID != "call_123" {
		t.Errorf("Expected ToolCallID 'call_123', got %q", delta.ToolCallID)
	}
	if delta.Name != "get_weather" {
		t.Errorf("Expected Name 'get_weather', got %q", delta.Name)
	}
	if delta.ArgsDelta != `{"location": "Seattle"}` {
		t.Errorf("Expected ArgsDelta '{\"location\": \"Seattle\"}', got %q", delta.ArgsDelta)
	}
}

func TestUsageDetails_AllFields(t *testing.T) {
	usage := &UsageDetails{
		InputTokens:     100,
		OutputTokens:    50,
		TotalTokens:     150,
		CachedTokens:    20,
		ReasoningTokens: 10,
	}

	if usage.InputTokens != 100 {
		t.Errorf("Expected InputTokens 100, got %d", usage.InputTokens)
	}
	if usage.OutputTokens != 50 {
		t.Errorf("Expected OutputTokens 50, got %d", usage.OutputTokens)
	}
	if usage.TotalTokens != 150 {
		t.Errorf("Expected TotalTokens 150, got %d", usage.TotalTokens)
	}
	if usage.CachedTokens != 20 {
		t.Errorf("Expected CachedTokens 20, got %d", usage.CachedTokens)
	}
	if usage.ReasoningTokens != 10 {
		t.Errorf("Expected ReasoningTokens 10, got %d", usage.ReasoningTokens)
	}
}

func TestAsyncRunStatusValues(t *testing.T) {
	// Verify status string values match expected API values
	expectedValues := map[AsyncRunStatus]string{
		StatusQueued:         "queued",
		StatusInProgress:     "in_progress",
		StatusRequiresAction: "requires_action",
		StatusCompleted:      "completed",
		StatusCanceled:       "canceled",
		StatusFailed:         "failed",
		StatusExpired:        "expired",
	}

	for status, expected := range expectedValues {
		if string(status) != expected {
			t.Errorf("Expected status %v to have value %q, got %q", status, expected, string(status))
		}
	}
}
