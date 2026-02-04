// Copyright (c) Microsoft. All rights reserved.

package a2a

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
)

// Client provides methods for interacting with A2A agents.
type Client struct {
	baseURL    string
	httpClient *http.Client
	headers    map[string]string
}

// ClientOption configures the A2A client.
type ClientOption func(*clientOptions)

type clientOptions struct {
	httpClient *http.Client
	headers    map[string]string
	timeout    time.Duration
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ClientOption {
	return func(o *clientOptions) {
		o.httpClient = client
	}
}

// WithHeader adds a header to all requests.
func WithHeader(key, value string) ClientOption {
	return func(o *clientOptions) {
		if o.headers == nil {
			o.headers = make(map[string]string)
		}
		o.headers[key] = value
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// NewClient creates a new A2A client.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	options := clientOptions{
		timeout: 30 * time.Second,
	}
	for _, opt := range opts {
		opt(&options)
	}

	httpClient := options.httpClient
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: options.timeout,
		}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		headers:    options.headers,
	}
}

// doRequest performs an HTTP request with common handling.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	for key, value := range c.headers {
		req.Header.Set(key, value)
	}

	return c.httpClient.Do(req)
}

// GetAgentCard retrieves the agent's capability card.
func (c *Client) GetAgentCard(ctx context.Context) (*AgentCard, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/.well-known/agent.json", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var card AgentCard
	if err := json.NewDecoder(resp.Body).Decode(&card); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &card, nil
}

// CreateTask creates a new task with the agent.
func (c *Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (*Task, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "/tasks", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

// GetTask retrieves a task by ID.
func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "/tasks/"+taskID, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

// SendMessage sends a message to a task.
func (c *Client) SendMessage(ctx context.Context, taskID string, msg *Message) (*Task, error) {
	req := &SendMessageRequest{Message: msg}
	resp, err := c.doRequest(ctx, http.MethodPost, "/tasks/"+taskID+"/messages", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &task, nil
}

// SendMessageStream sends a message and streams the response.
func (c *Client) SendMessageStream(ctx context.Context, taskID string, msg *Message) (<-chan StreamEvent, error) {
	req := &SendMessageRequest{Message: msg}
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/tasks/"+taskID+"/messages/stream",
		bytes.NewReader(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	events := make(chan StreamEvent, 100)
	go func() {
		defer close(events)
		defer resp.Body.Close()
		c.parseSSE(ctx, resp.Body, events)
	}()

	return events, nil
}

// parseSSE parses Server-Sent Events from a reader.
func (c *Client) parseSSE(ctx context.Context, r io.Reader, events chan<- StreamEvent) {
	scanner := bufio.NewScanner(r)
	var eventType string
	var dataLines []string

	for scanner.Scan() {
		if ctx.Err() != nil {
			events <- StreamEvent{
				Type:  StreamEventTypeError,
				Error: ctx.Err(),
			}
			return
		}

		line := scanner.Text()

		if line == "" {
			// End of event - process accumulated data
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				event := StreamEvent{
					Type: eventType,
				}

				// Parse the data based on event type
				if eventType == StreamEventTypeMessage {
					var msg Message
					if err := json.Unmarshal([]byte(data), &msg); err == nil {
						event.Message = &msg
					}
				} else if eventType == StreamEventTypeTask {
					var task Task
					if err := json.Unmarshal([]byte(data), &task); err == nil {
						event.Task = &task
					}
				} else {
					// Store raw data for other event types
					event.Data = json.RawMessage(data)
				}

				events <- event
			}
			eventType = ""
			dataLines = nil
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
		}
	}

	if err := scanner.Err(); err != nil {
		events <- StreamEvent{
			Type:  StreamEventTypeError,
			Error: err,
		}
	}
}

// CancelTask cancels a running task.
func (c *Client) CancelTask(ctx context.Context, taskID string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "/tasks/"+taskID, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status: %s", resp.Status)
	}

	return nil
}
