// Copyright (c) Microsoft. All rights reserved.

package agui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
)

// Client communicates with AG-UI protocol servers.
// It sends requests and parses SSE event streams, converting them to
// framework-native ResponseUpdate types.
type Client struct {
	endpoint   string
	httpClient *http.Client
	options    clientOptions
}

type clientOptions struct {
	timeout    time.Duration
	headers    map[string]string
	httpClient *http.Client
}

// ClientOption configures the AG-UI client.
type ClientOption func(*clientOptions)

// WithTimeout sets the HTTP request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithHeader adds a custom header to all requests.
func WithHeader(key, value string) ClientOption {
	return func(o *clientOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(o *clientOptions) {
		o.httpClient = client
	}
}

// NewClient creates a new AG-UI client for the given endpoint.
func NewClient(endpoint string, opts ...ClientOption) *Client {
	options := clientOptions{
		timeout: 60 * time.Second,
		headers: make(map[string]string),
	}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: options.timeout}
	}

	return &Client{
		endpoint:   endpoint,
		httpClient: httpClient,
		options:    options,
	}
}

// ClientRunRequest contains parameters for an AG-UI run.
type ClientRunRequest struct {
	// ThreadID identifies the conversation thread
	ThreadID string

	// RunID identifies the specific run within the thread
	RunID string

	// Messages are the input messages to send
	Messages []agent.Message

	// State is additional state to send with the request
	State map[string]interface{}

	// Tools is an optional list of tools to send with the request
	Tools []interface{}

	// Context is an optional list of context items to send with the request
	Context []interface{}

	// ForwardedProps are optional forwarded properties for the request
	ForwardedProps map[string]interface{}
}

// Run executes a request and returns the complete response.
// This is a convenience method that collects all streaming updates.
func (c *Client) Run(ctx context.Context, req *ClientRunRequest) (*agent.Response, error) {
	updates, err := c.RunStream(ctx, req)
	if err != nil {
		return nil, err
	}

	// Collect all updates into a response
	var response agent.Response
	var messages []agent.Message
	var currentMessage *agent.Message

	for update := range updates {
		switch update.Kind {
		case agent.UpdateKindMessageComplete:
			if update.Message != nil {
				messages = append(messages, *update.Message)
			}
		case agent.UpdateKindContentDelta:
			if update.Delta != nil && update.Delta.TextDelta != "" {
				// Build message incrementally
				if currentMessage == nil {
					currentMessage = &agent.Message{Role: chat.RoleAssistant}
				}
				// Accumulate text delta - full message will be in MessageComplete
			}
		case agent.UpdateKindError:
			if update.Error != nil {
				return nil, update.Error
			}
		}
	}

	response.Messages = messages
	return &response, nil
}

// RunStream executes a request and returns a channel of response updates.
// The channel is closed when the stream ends or an error occurs.
func (c *Client) RunStream(ctx context.Context, req *ClientRunRequest) (<-chan agent.ResponseUpdate, error) {
	// Build request body
	body := c.buildRequestBody(req)

	// Serialize request body
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	for k, v := range c.options.headers {
		httpReq.Header.Set(k, v)
	}

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Convert SSE events to ResponseUpdates
	updates := make(chan agent.ResponseUpdate, 100)

	go c.processStream(ctx, resp.Body, updates)

	return updates, nil
}

// processStream reads SSE events and converts them to ResponseUpdates.
func (c *Client) processStream(ctx context.Context, r io.ReadCloser, updates chan<- agent.ResponseUpdate) {
	defer close(updates)
	defer r.Close()

	scanner := bufio.NewScanner(r)
	converter := newResponseConverter()

	var eventType string
	var dataLines []string

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		line := scanner.Text()

		if line == "" {
			// Empty line = event boundary
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				event, err := c.parseEvent(eventType, data)
				if err == nil && event != nil {
					for _, update := range converter.Convert(event) {
						select {
						case updates <- update:
						case <-ctx.Done():
							return
						}
					}
				}
				eventType = ""
				dataLines = nil
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		}
		// Ignore comments (lines starting with :) and id: lines for now
	}

	if err := scanner.Err(); err != nil {
		updates <- agent.ResponseUpdate{
			Kind:  agent.UpdateKindError,
			Error: err,
		}
	}
}

// parseEvent converts raw SSE data to a typed AG-UI event.
func (c *Client) parseEvent(eventType, data string) (Event, error) {
	// Map event type string to EventType constant
	var evtType EventType
	switch strings.ToUpper(eventType) {
	case "RUN_STARTED":
		evtType = EventTypeRunStarted
	case "RUN_FINISHED":
		evtType = EventTypeRunFinished
	case "RUN_ERROR":
		evtType = EventTypeRunError
	case "TEXT_MESSAGE_START":
		evtType = EventTypeTextMessageStart
	case "TEXT_MESSAGE_CONTENT":
		evtType = EventTypeTextMessageContent
	case "TEXT_MESSAGE_END":
		evtType = EventTypeTextMessageEnd
	case "TOOL_CALL_START":
		evtType = EventTypeToolCallStart
	case "TOOL_CALL_ARGS":
		evtType = EventTypeToolCallArgs
	case "TOOL_CALL_END":
		evtType = EventTypeToolCallEnd
	case "TOOL_CALL_RESULT":
		evtType = EventTypeToolCallResult
	default:
		// Try to parse from data if event field is empty or unknown
		var typeWrapper struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(data), &typeWrapper); err == nil {
			evtType = EventType(typeWrapper.Type)
		} else {
			return nil, nil // Unknown event, skip
		}
	}

	// Parse based on event type
	switch evtType {
	case EventTypeRunStarted:
		var e RunStartedEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse RunStartedEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeRunStarted}
		return &e, nil

	case EventTypeRunFinished:
		var e RunFinishedEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse RunFinishedEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeRunFinished}
		return &e, nil

	case EventTypeTextMessageStart:
		var e TextMessageStartEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse TextMessageStartEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeTextMessageStart}
		return &e, nil

	case EventTypeTextMessageContent:
		var e TextMessageContentEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse TextMessageContentEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeTextMessageContent}
		return &e, nil

	case EventTypeTextMessageEnd:
		var e TextMessageEndEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse TextMessageEndEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeTextMessageEnd}
		return &e, nil

	case EventTypeToolCallStart:
		var e ToolCallStartEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse ToolCallStartEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeToolCallStart}
		return &e, nil

	case EventTypeToolCallArgs:
		var e ToolCallArgsEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse ToolCallArgsEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeToolCallArgs}
		return &e, nil

	case EventTypeToolCallEnd:
		var e ToolCallEndEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse ToolCallEndEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeToolCallEnd}
		return &e, nil

	case EventTypeRunError:
		var e RunErrorEvent
		if err := json.Unmarshal([]byte(data), &e); err != nil {
			return nil, fmt.Errorf("failed to parse RunErrorEvent: %w", err)
		}
		e.eventBase = eventBase{eventType: EventTypeRunError}
		return &e, nil

	default:
		return nil, nil // Unknown event type, skip
	}
}

// buildRequestBody creates the JSON request body.
func (c *Client) buildRequestBody(req *ClientRunRequest) map[string]interface{} {
	body := map[string]interface{}{
		"threadId": req.ThreadID,
	}

	if req.RunID != "" {
		body["runId"] = req.RunID
	}

	if len(req.Messages) > 0 {
		// Convert messages to request format
		msgs := make([]map[string]interface{}, len(req.Messages))
		for i, m := range req.Messages {
			msgs[i] = map[string]interface{}{
				"role":    string(m.Role),
				"content": m.Text(),
			}
		}
		body["messages"] = msgs
	}

	if len(req.State) > 0 {
		body["state"] = req.State
	}

	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}

	if len(req.Context) > 0 {
		body["context"] = req.Context
	}

	if len(req.ForwardedProps) > 0 {
		body["forwardedProps"] = req.ForwardedProps
	}

	return body
}

// responseConverter converts AG-UI events to agent ResponseUpdates.
type responseConverter struct {
	// activeMessage tracks the current message being built
	activeMessage *messageBuilder

	// activeToolCalls tracks tool calls being built
	activeToolCalls map[string]*toolCallBuilder
}

type messageBuilder struct {
	id      string
	role    string
	content string
}

type toolCallBuilder struct {
	id   string
	name string
	args string
}

// newResponseConverter creates a new response converter.
func newResponseConverter() *responseConverter {
	return &responseConverter{
		activeToolCalls: make(map[string]*toolCallBuilder),
	}
}

// Convert transforms an AG-UI event into zero or more ResponseUpdates.
func (c *responseConverter) Convert(event Event) []agent.ResponseUpdate {
	switch e := event.(type) {
	case *RunStartedEvent:
		return []agent.ResponseUpdate{
			{Kind: agent.UpdateKindDone}, // Use as "started" signal
		}

	case *RunFinishedEvent:
		return []agent.ResponseUpdate{
			{Kind: agent.UpdateKindDone},
		}

	case *TextMessageStartEvent:
		c.activeMessage = &messageBuilder{
			id:   e.MessageID,
			role: e.Role,
		}
		return nil

	case *TextMessageContentEvent:
		if c.activeMessage != nil {
			c.activeMessage.content += e.Delta
		}
		return []agent.ResponseUpdate{
			{
				Kind: agent.UpdateKindContentDelta,
				Delta: &agent.ContentDelta{
					TextDelta: e.Delta,
				},
			},
		}

	case *TextMessageEndEvent:
		if c.activeMessage != nil {
			var msg agent.Message
			// Add text content to message
			if c.activeMessage.content != "" {
				if c.activeMessage.role == "assistant" {
					msg = agent.NewAssistantMessage(c.activeMessage.content)
				} else if c.activeMessage.role == "user" {
					msg = agent.NewUserMessage(c.activeMessage.content)
				} else {
					msg = agent.NewSystemMessage(c.activeMessage.content)
				}
			} else {
				msg = agent.Message{Role: chat.Role(c.activeMessage.role)}
			}
			c.activeMessage = nil
			return []agent.ResponseUpdate{
				{
					Kind:    agent.UpdateKindMessageComplete,
					Message: &msg,
				},
			}
		}
		return nil

	case *ToolCallStartEvent:
		c.activeToolCalls[e.ToolCallID] = &toolCallBuilder{
			id:   e.ToolCallID,
			name: e.ToolCallName,
		}
		return []agent.ResponseUpdate{
			{
				Kind: agent.UpdateKindToolCall,
				Delta: &agent.ContentDelta{
					ToolCallID: e.ToolCallID,
					Name:       e.ToolCallName,
				},
			},
		}

	case *ToolCallArgsEvent:
		if tc, ok := c.activeToolCalls[e.ToolCallID]; ok {
			tc.args += e.Delta
		}
		return []agent.ResponseUpdate{
			{
				Kind: agent.UpdateKindContentDelta,
				Delta: &agent.ContentDelta{
					ToolCallID: e.ToolCallID,
					ArgsDelta:  e.Delta,
				},
			},
		}

	case *ToolCallEndEvent:
		if tc, ok := c.activeToolCalls[e.ToolCallID]; ok {
			delete(c.activeToolCalls, e.ToolCallID)
			return []agent.ResponseUpdate{
				{
					Kind: agent.UpdateKindToolCall,
					Delta: &agent.ContentDelta{
						ToolCallID: tc.id,
						Name:       tc.name,
						ArgsDelta:  tc.args,
					},
				},
			}
		}
		return nil

	case *RunErrorEvent:
		return []agent.ResponseUpdate{
			{
				Kind:  agent.UpdateKindError,
				Error: fmt.Errorf("%s", e.Message),
			},
		}

	default:
		return nil
	}
}
