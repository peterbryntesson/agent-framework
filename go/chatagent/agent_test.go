// Copyright (c) Microsoft. All rights reserved.

package chatagent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/microsoft/agent-framework-go/agent"
	"github.com/microsoft/agent-framework-go/chat"
	"github.com/microsoft/agent-framework-go/tool"
)

// mockClient is a mock implementation of chat.Client for testing.
type mockClient struct {
	metadata       chat.ClientMetadata
	responses      []*chat.Response
	responseIndex  int
	streamUpdates  [][]chat.ResponseUpdate
	streamIndex    int
	err            error
	getResponseFn  func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error)
	getStreamingFn func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error)
}

func newMockClient() *mockClient {
	return &mockClient{
		metadata: chat.ClientMetadata{
			ProviderName: "mock",
			ModelID:      "mock-model",
			EndpointURI:  "http://mock.example.com",
		},
	}
}

func (m *mockClient) GetResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
	if m.getResponseFn != nil {
		return m.getResponseFn(ctx, messages, options)
	}
	if m.err != nil {
		return nil, m.err
	}
	if m.responseIndex < len(m.responses) {
		resp := m.responses[m.responseIndex]
		m.responseIndex++
		return resp, nil
	}
	return &chat.Response{
		Message: chat.NewAssistantMessage("Mock response"),
		Usage:   &chat.UsageDetails{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
	}, nil
}

func (m *mockClient) GetStreamingResponse(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
	if m.getStreamingFn != nil {
		return m.getStreamingFn(ctx, messages, options)
	}
	if m.err != nil {
		return nil, m.err
	}

	updates := make(chan chat.ResponseUpdate, 16)
	go func() {
		defer close(updates)

		if m.streamIndex < len(m.streamUpdates) {
			for _, update := range m.streamUpdates[m.streamIndex] {
				select {
				case <-ctx.Done():
					return
				case updates <- update:
				}
			}
			m.streamIndex++
		} else {
			// Default stream
			updates <- chat.ResponseUpdate{
				Kind:  chat.UpdateKindContentDelta,
				Delta: &chat.ContentDelta{TextDelta: "Mock "},
			}
			updates <- chat.ResponseUpdate{
				Kind:  chat.UpdateKindContentDelta,
				Delta: &chat.ContentDelta{TextDelta: "response"},
			}
			updates <- chat.ResponseUpdate{
				Kind:         chat.UpdateKindDone,
				FinishReason: chat.FinishReasonStop,
			}
		}
	}()

	return updates, nil
}

func (m *mockClient) Metadata() chat.ClientMetadata {
	return m.metadata
}

// TestNewAgent tests agent creation.
func TestNewAgent(t *testing.T) {
	client := newMockClient()

	t.Run("creates agent with defaults", func(t *testing.T) {
		a := New(client)

		if a.ID() == "" {
			t.Error("expected generated ID")
		}
		if a.Name() != "mock-model" {
			t.Errorf("expected name 'mock-model', got %q", a.Name())
		}
		if a.Description() != "" {
			t.Errorf("expected empty description, got %q", a.Description())
		}
		if a.Client() != client {
			t.Error("expected client to be stored")
		}
	})

	t.Run("creates agent with custom options", func(t *testing.T) {
		a := New(client,
			WithID("custom-id"),
			WithName("MyAgent"),
			WithDescription("Test agent"),
			WithInstructions("Be helpful"),
			WithMaxTurns(20),
		)

		if a.ID() != "custom-id" {
			t.Errorf("expected ID 'custom-id', got %q", a.ID())
		}
		if a.Name() != "MyAgent" {
			t.Errorf("expected name 'MyAgent', got %q", a.Name())
		}
		if a.Description() != "Test agent" {
			t.Errorf("expected description 'Test agent', got %q", a.Description())
		}
		if a.Instructions() != "Be helpful" {
			t.Errorf("expected instructions 'Be helpful', got %q", a.Instructions())
		}
	})

	t.Run("creates agent with tools", func(t *testing.T) {
		mockTool := &mockFunctionTool{name: "test_tool"}
		a := New(client, WithTools(mockTool))

		if len(a.Tools()) != 1 {
			t.Errorf("expected 1 tool, got %d", len(a.Tools()))
		}
		if a.Tools()[0].Name() != "test_tool" {
			t.Errorf("expected tool name 'test_tool', got %q", a.Tools()[0].Name())
		}
	})
}

// TestAgentImplementsInterface verifies Agent implements agent.Agent.
func TestAgentImplementsInterface(t *testing.T) {
	client := newMockClient()
	a := New(client)

	var _ agent.Agent = a
}

// TestAgentMetadata tests metadata retrieval.
func TestAgentMetadata(t *testing.T) {
	client := newMockClient()
	a := New(client)

	meta := a.Metadata()

	if meta.ProviderName != "mock" {
		t.Errorf("expected provider 'mock', got %q", meta.ProviderName)
	}
}

// TestAgentRun tests the Run method.
func TestAgentRun(t *testing.T) {
	t.Run("basic run returns response", func(t *testing.T) {
		client := newMockClient()
		a := New(client)

		resp, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("Hello"),
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp == nil {
			t.Fatal("expected response")
		}
		if resp.Text() != "Mock response" {
			t.Errorf("expected 'Mock response', got %q", resp.Text())
		}
	})

	t.Run("run with instructions prepends system message", func(t *testing.T) {
		client := newMockClient()
		var capturedMessages []chat.Message

		client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			capturedMessages = messages
			return &chat.Response{
				Message: chat.NewAssistantMessage("Response"),
				Usage:   &chat.UsageDetails{},
			}, nil
		}

		a := New(client, WithInstructions("Be helpful"))
		_, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("Hello"),
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(capturedMessages) != 2 {
			t.Fatalf("expected 2 messages, got %d", len(capturedMessages))
		}
		if capturedMessages[0].Role != chat.RoleSystem {
			t.Errorf("expected system message first, got %v", capturedMessages[0].Role)
		}
		if capturedMessages[0].Text() != "Be helpful" {
			t.Errorf("expected 'Be helpful', got %q", capturedMessages[0].Text())
		}
	})

	t.Run("run with session includes history", func(t *testing.T) {
		client := newMockClient()
		var capturedMessages []chat.Message

		client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			capturedMessages = messages
			return &chat.Response{
				Message: chat.NewAssistantMessage("Response"),
				Usage:   &chat.UsageDetails{},
			}, nil
		}

		a := New(client, WithInstructions("Be helpful"))
		session, _ := a.NewSession(context.Background())
		session.AddMessage(agent.NewUserMessage("First message"))
		session.AddMessage(agent.NewAssistantMessage("First response"))

		_, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("Second message"),
		}, agent.WithSession(session))

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should have: system + 2 history + 1 new = 4 messages
		if len(capturedMessages) != 4 {
			t.Errorf("expected 4 messages, got %d", len(capturedMessages))
		}
	})

	t.Run("run returns error from client", func(t *testing.T) {
		client := newMockClient()
		client.err = errors.New("client error")

		a := New(client)
		_, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("Hello"),
		})

		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, client.err) && err.Error() != "chat request failed: client error" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("run respects context cancellation", func(t *testing.T) {
		client := newMockClient()
		client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return &chat.Response{
					Message: chat.NewAssistantMessage("Response"),
					Usage:   &chat.UsageDetails{},
				}, nil
			}
		}

		a := New(client)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		_, err := a.Run(ctx, []agent.Message{
			agent.NewUserMessage("Hello"),
		})

		if err == nil {
			t.Fatal("expected error from cancelled context")
		}
	})
}

// TestAgentRunStream tests the RunStream method.
func TestAgentRunStream(t *testing.T) {
	t.Run("streams content updates", func(t *testing.T) {
		client := newMockClient()
		a := New(client)

		updates, err := a.RunStream(context.Background(), []agent.Message{
			agent.NewUserMessage("Hello"),
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var text string
		for update := range updates {
			if update.Kind == agent.UpdateKindContentDelta && update.Delta != nil {
				text += update.Delta.TextDelta
			}
		}

		if text != "Mock response" {
			t.Errorf("expected 'Mock response', got %q", text)
		}
	})

	t.Run("stream returns error through channel", func(t *testing.T) {
		client := newMockClient()
		client.getStreamingFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (<-chan chat.ResponseUpdate, error) {
			return nil, errors.New("stream error")
		}

		a := New(client)
		updates, err := a.RunStream(context.Background(), []agent.Message{
			agent.NewUserMessage("Hello"),
		})

		// RunStream itself may not return error immediately since it starts a goroutine
		// If no error returned, check the channel for error update
		if err != nil {
			return // Error returned directly, test passes
		}

		// Consume updates looking for error
		foundError := false
		for update := range updates {
			if update.Kind == agent.UpdateKindError {
				foundError = true
				break
			}
		}

		if !foundError {
			t.Fatal("expected error update in channel")
		}
	})
}

// TestAgentWithTools tests tool integration.
func TestAgentWithTools(t *testing.T) {
	t.Run("tool definitions sent to client", func(t *testing.T) {
		client := newMockClient()
		var capturedOptions *chat.Options

		client.getResponseFn = func(ctx context.Context, messages []chat.Message, options *chat.Options) (*chat.Response, error) {
			capturedOptions = options
			return &chat.Response{
				Message: chat.NewAssistantMessage("Response"),
				Usage:   &chat.UsageDetails{},
			}, nil
		}

		mockTool := &mockFunctionTool{
			name:        "get_weather",
			description: "Get weather for a location",
		}
		a := New(client, WithTools(mockTool))

		_, err := a.Run(context.Background(), []agent.Message{
			agent.NewUserMessage("What's the weather?"),
		})

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedOptions == nil {
			t.Fatal("expected options to be set")
		}
		if len(capturedOptions.Tools) != 1 {
			t.Fatalf("expected 1 tool, got %d", len(capturedOptions.Tools))
		}
		if capturedOptions.Tools[0].Name != "get_weather" {
			t.Errorf("expected tool name 'get_weather', got %q", capturedOptions.Tools[0].Name)
		}
	})
}

// TestAgentServices tests service registration and retrieval.
func TestAgentServices(t *testing.T) {
	type TestService struct {
		Value string
	}

	client := newMockClient()
	a := New(client)

	service := &TestService{Value: "test"}
	a.RegisterService(service)

	retrieved, ok := agent.GetService[*TestService](a)
	if !ok {
		t.Fatal("expected to retrieve service")
	}
	if retrieved.Value != "test" {
		t.Errorf("expected value 'test', got %q", retrieved.Value)
	}
}

// mockFunctionTool is a mock implementation of tool.Tool.
type mockFunctionTool struct {
	name         string
	description  string
	parameters   json.RawMessage
	invokeResult tool.Result
	invokeErr    error
}

func (m *mockFunctionTool) Name() string                { return m.name }
func (m *mockFunctionTool) Description() string         { return m.description }
func (m *mockFunctionTool) Parameters() json.RawMessage { return m.parameters }
func (m *mockFunctionTool) Invoke(ctx context.Context, args json.RawMessage) (tool.Result, error) {
	if m.invokeErr != nil {
		return tool.Result{}, m.invokeErr
	}
	return m.invokeResult, nil
}
