// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"time"
)

// AgentInfo describes a registered agent.
type AgentInfo struct {
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// RunRequest is the request body for running an agent.
type RunRequest struct {
	Messages      []MessageInput `json:"messages"`
	ConversationID string        `json:"conversationId,omitempty"`
}

// MessageInput represents an input message.
type MessageInput struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RunResponse is the response from a non-streaming run.
type RunResponse struct {
	ID             string         `json:"id"`
	ConversationID string         `json:"conversationId"`
	Content        string         `json:"content"`
	FinishReason   string         `json:"finishReason,omitempty"`
	Usage          *UsageInfo     `json:"usage,omitempty"`
	TraceID        string         `json:"traceId,omitempty"`
}

// UsageInfo contains token usage information.
type UsageInfo struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
	TotalTokens  int `json:"totalTokens"`
}

// StreamEvent represents a server-sent event during streaming.
type StreamEvent struct {
	Event string `json:"event"`
	Data  string `json:"data"`
}

// TraceInfo describes a collected trace.
type TraceInfo struct {
	TraceID    string        `json:"traceId"`
	AgentName  string        `json:"agentName"`
	StartTime  time.Time     `json:"startTime"`
	EndTime    time.Time     `json:"endTime,omitempty"`
	Duration   time.Duration `json:"duration,omitempty"`
	Status     string        `json:"status"`
	SpanCount  int           `json:"spanCount"`
}

// TraceDetail provides full details of a trace including all spans.
type TraceDetail struct {
	TraceInfo
	Spans []SpanInfo `json:"spans"`
}

// SpanInfo describes a single span within a trace.
type SpanInfo struct {
	SpanID       string            `json:"spanId"`
	ParentSpanID string            `json:"parentSpanId,omitempty"`
	Name         string            `json:"name"`
	StartTime    time.Time         `json:"startTime"`
	EndTime      time.Time         `json:"endTime"`
	Duration     time.Duration     `json:"duration"`
	Status       string            `json:"status"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Events       []SpanEvent       `json:"events,omitempty"`
}

// SpanEvent represents an event recorded during a span.
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// ErrorResponse represents an API error.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// AgentsResponse is the response for listing agents.
type AgentsResponse struct {
	Agents []AgentInfo `json:"agents"`
}

// TracesResponse is the response for listing traces.
type TracesResponse struct {
	Traces []TraceInfo `json:"traces"`
	Total  int         `json:"total"`
}
