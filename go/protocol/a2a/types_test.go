// Copyright (c) Microsoft. All rights reserved.

package a2a

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentCard_JSONSerialization(t *testing.T) {
	t.Run("serializes complete card", func(t *testing.T) {
		// Arrange
		card := AgentCard{
			Name:        "Test Agent",
			Description: "A test agent",
			URL:         "https://agent.example.com",
			Provider: &AgentProvider{
				Name: "Test Corp",
				URL:  "https://test.com",
			},
			Version: "1.0.0",
			Capabilities: &AgentCapabilities{
				Streaming:         true,
				PushNotifications: false,
				StateTransitions:  []string{"pending", "running", "completed"},
			},
			Authentication: &AgentAuthentication{
				Schemes: []string{"bearer", "api_key"},
			},
			Skills: []AgentSkill{
				{ID: "skill-1", Name: "Search", Description: "Search capability", Tags: []string{"search", "web"}},
			},
		}

		// Act
		data, err := json.Marshal(card)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"name":"Test Agent"`)
		assert.Contains(t, string(data), `"streaming":true`)
		assert.Contains(t, string(data), `"skills":[`)
	})

	t.Run("deserializes complete card", func(t *testing.T) {
		// Arrange
		jsonData := `{
			"name": "Test Agent",
			"description": "A test agent",
			"url": "https://agent.example.com",
			"provider": {"name": "Test Corp"},
			"version": "1.0.0",
			"capabilities": {"streaming": true},
			"skills": [{"id": "s1", "name": "Skill 1"}]
		}`

		// Act
		var card AgentCard
		err := json.Unmarshal([]byte(jsonData), &card)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "Test Agent", card.Name)
		assert.Equal(t, "A test agent", card.Description)
		assert.Equal(t, "https://agent.example.com", card.URL)
		assert.Equal(t, "Test Corp", card.Provider.Name)
		assert.True(t, card.Capabilities.Streaming)
		assert.Len(t, card.Skills, 1)
		assert.Equal(t, "s1", card.Skills[0].ID)
	})

	t.Run("omits empty optional fields", func(t *testing.T) {
		// Arrange
		card := AgentCard{
			Name: "Minimal Agent",
			URL:  "https://minimal.example.com",
		}

		// Act
		data, err := json.Marshal(card)

		// Assert
		require.NoError(t, err)
		assert.NotContains(t, string(data), `"description"`)
		assert.NotContains(t, string(data), `"provider"`)
		assert.NotContains(t, string(data), `"capabilities"`)
	})
}

func TestTaskState_Constants(t *testing.T) {
	t.Run("states have correct values", func(t *testing.T) {
		// Assert
		assert.Equal(t, TaskState("pending"), TaskStatePending)
		assert.Equal(t, TaskState("running"), TaskStateRunning)
		assert.Equal(t, TaskState("completed"), TaskStateCompleted)
		assert.Equal(t, TaskState("failed"), TaskStateFailed)
		assert.Equal(t, TaskState("cancelled"), TaskStateCancelled)
	})
}

func TestTask_JSONSerialization(t *testing.T) {
	t.Run("serializes complete task", func(t *testing.T) {
		// Arrange
		now := time.Now().Truncate(time.Second)
		task := Task{
			ID:        "task-123",
			ContextID: "ctx-456",
			State:     TaskStateRunning,
			Messages: []Message{
				{Role: RoleUser, Parts: []Part{NewTextPart("Hello")}},
			},
			Artifacts: []Artifact{
				{ID: "art-1", Name: "Result", Parts: []Part{NewTextPart("Output")}},
			},
			Metadata:  map[string]interface{}{"key": "value"},
			CreatedAt: now,
			UpdatedAt: now,
		}

		// Act
		data, err := json.Marshal(task)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"id":"task-123"`)
		assert.Contains(t, string(data), `"state":"running"`)
		assert.Contains(t, string(data), `"context_id":"ctx-456"`)
	})

	t.Run("deserializes task with error", func(t *testing.T) {
		// Arrange
		jsonData := `{
			"id": "task-789",
			"state": "failed",
			"error": {"code": "EXECUTION_ERROR", "message": "Something went wrong"}
		}`

		// Act
		var task Task
		err := json.Unmarshal([]byte(jsonData), &task)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "task-789", task.ID)
		assert.Equal(t, TaskStateFailed, task.State)
		require.NotNil(t, task.Error)
		assert.Equal(t, "EXECUTION_ERROR", task.Error.Code)
		assert.Equal(t, "Something went wrong", task.Error.Message)
	})
}

func TestCreateTaskRequest_JSONSerialization(t *testing.T) {
	t.Run("serializes request", func(t *testing.T) {
		// Arrange
		msg := NewUserMessage("Hello")
		req := CreateTaskRequest{
			ContextID: "ctx-123",
			Message:   &msg,
			Metadata:  map[string]interface{}{"priority": "high"},
		}

		// Act
		data, err := json.Marshal(req)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"context_id":"ctx-123"`)
		assert.Contains(t, string(data), `"message":{`)
		assert.Contains(t, string(data), `"priority":"high"`)
	})
}

func TestMessageRole_Constants(t *testing.T) {
	t.Run("roles have correct values", func(t *testing.T) {
		// Assert
		assert.Equal(t, MessageRole("user"), RoleUser)
		assert.Equal(t, MessageRole("assistant"), RoleAssistant)
		assert.Equal(t, MessageRole("system"), RoleSystem)
	})
}

func TestMessage_JSONSerialization(t *testing.T) {
	t.Run("serializes message with parts", func(t *testing.T) {
		// Arrange
		msg := Message{
			Role: RoleUser,
			Parts: []Part{
				NewTextPart("Hello"),
				NewDataPart(map[string]string{"key": "value"}, "application/json"),
			},
			Metadata: map[string]interface{}{"source": "test"},
		}

		// Act
		data, err := json.Marshal(msg)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"role":"user"`)
		assert.Contains(t, string(data), `"type":"text"`)
		assert.Contains(t, string(data), `"type":"data"`)
	})

	t.Run("deserializes message", func(t *testing.T) {
		// Arrange
		jsonData := `{
			"role": "assistant",
			"parts": [
				{"type": "text", "text": "Hello, world!"},
				{"type": "file", "uri": "https://example.com/file.pdf", "mime_type": "application/pdf"}
			]
		}`

		// Act
		var msg Message
		err := json.Unmarshal([]byte(jsonData), &msg)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, RoleAssistant, msg.Role)
		require.Len(t, msg.Parts, 2)
		assert.Equal(t, PartTypeText, msg.Parts[0].Type)
		assert.Equal(t, "Hello, world!", msg.Parts[0].Text)
		assert.Equal(t, PartTypeFile, msg.Parts[1].Type)
		assert.Equal(t, "https://example.com/file.pdf", msg.Parts[1].URI)
	})
}

func TestMessage_Text(t *testing.T) {
	t.Run("returns concatenated text from text parts", func(t *testing.T) {
		// Arrange
		msg := Message{
			Role: RoleUser,
			Parts: []Part{
				NewTextPart("Hello, "),
				NewDataPart("ignored", "application/json"),
				NewTextPart("world!"),
			},
		}

		// Act
		text := msg.Text()

		// Assert
		assert.Equal(t, "Hello, world!", text)
	})

	t.Run("returns empty string for no text parts", func(t *testing.T) {
		// Arrange
		msg := Message{
			Role: RoleUser,
			Parts: []Part{
				NewDataPart("data", "application/json"),
			},
		}

		// Act
		text := msg.Text()

		// Assert
		assert.Equal(t, "", text)
	})

	t.Run("returns empty string for empty parts", func(t *testing.T) {
		// Arrange
		msg := Message{Role: RoleUser}

		// Act
		text := msg.Text()

		// Assert
		assert.Equal(t, "", text)
	})
}

func TestPartType_Constants(t *testing.T) {
	t.Run("part types have correct values", func(t *testing.T) {
		// Assert
		assert.Equal(t, PartType("text"), PartTypeText)
		assert.Equal(t, PartType("data"), PartTypeData)
		assert.Equal(t, PartType("file"), PartTypeFile)
	})
}

func TestPart_Constructors(t *testing.T) {
	t.Run("NewTextPart creates text part", func(t *testing.T) {
		// Act
		part := NewTextPart("Hello")

		// Assert
		assert.Equal(t, PartTypeText, part.Type)
		assert.Equal(t, "Hello", part.Text)
	})

	t.Run("NewDataPart creates data part", func(t *testing.T) {
		// Arrange
		data := map[string]int{"count": 42}

		// Act
		part := NewDataPart(data, "application/json")

		// Assert
		assert.Equal(t, PartTypeData, part.Type)
		assert.Equal(t, data, part.Data)
		assert.Equal(t, "application/json", part.MimeType)
	})

	t.Run("NewFilePart creates file part", func(t *testing.T) {
		// Act
		part := NewFilePart("https://example.com/doc.pdf", "application/pdf")

		// Assert
		assert.Equal(t, PartTypeFile, part.Type)
		assert.Equal(t, "https://example.com/doc.pdf", part.URI)
		assert.Equal(t, "application/pdf", part.MimeType)
	})
}

func TestNewUserMessage(t *testing.T) {
	t.Run("creates user message with text", func(t *testing.T) {
		// Act
		msg := NewUserMessage("Hello")

		// Assert
		assert.Equal(t, RoleUser, msg.Role)
		require.Len(t, msg.Parts, 1)
		assert.Equal(t, PartTypeText, msg.Parts[0].Type)
		assert.Equal(t, "Hello", msg.Parts[0].Text)
		assert.False(t, msg.CreatedAt.IsZero())
	})
}

func TestNewAssistantMessage(t *testing.T) {
	t.Run("creates assistant message with text", func(t *testing.T) {
		// Act
		msg := NewAssistantMessage("Response")

		// Assert
		assert.Equal(t, RoleAssistant, msg.Role)
		require.Len(t, msg.Parts, 1)
		assert.Equal(t, PartTypeText, msg.Parts[0].Type)
		assert.Equal(t, "Response", msg.Parts[0].Text)
		assert.False(t, msg.CreatedAt.IsZero())
	})
}

func TestArtifact_JSONSerialization(t *testing.T) {
	t.Run("serializes artifact", func(t *testing.T) {
		// Arrange
		artifact := Artifact{
			ID:          "art-123",
			Name:        "Generated Code",
			Description: "Python code generated by the agent",
			Parts: []Part{
				NewTextPart("def hello():\n    print('Hello!')"),
			},
			Metadata:  map[string]interface{}{"language": "python"},
			CreatedAt: time.Now().Truncate(time.Second),
		}

		// Act
		data, err := json.Marshal(artifact)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"id":"art-123"`)
		assert.Contains(t, string(data), `"name":"Generated Code"`)
		assert.Contains(t, string(data), `"description":"Python code generated by the agent"`)
	})

	t.Run("deserializes artifact", func(t *testing.T) {
		// Arrange
		jsonData := `{
			"id": "art-456",
			"name": "Result",
			"parts": [{"type": "text", "text": "content"}],
			"metadata": {"format": "markdown"}
		}`

		// Act
		var artifact Artifact
		err := json.Unmarshal([]byte(jsonData), &artifact)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "art-456", artifact.ID)
		assert.Equal(t, "Result", artifact.Name)
		require.Len(t, artifact.Parts, 1)
		assert.Equal(t, "content", artifact.Parts[0].Text)
		assert.Equal(t, "markdown", artifact.Metadata["format"])
	})
}

func TestNewArtifact(t *testing.T) {
	t.Run("creates artifact with parts", func(t *testing.T) {
		// Act
		artifact := NewArtifact("art-1",
			NewTextPart("Part 1"),
			NewTextPart("Part 2"),
		)

		// Assert
		assert.Equal(t, "art-1", artifact.ID)
		require.Len(t, artifact.Parts, 2)
		assert.Equal(t, "Part 1", artifact.Parts[0].Text)
		assert.Equal(t, "Part 2", artifact.Parts[1].Text)
		assert.False(t, artifact.CreatedAt.IsZero())
	})
}

func TestSendMessageRequest_JSONSerialization(t *testing.T) {
	t.Run("serializes request", func(t *testing.T) {
		// Arrange
		msg := NewUserMessage("Test message")
		req := SendMessageRequest{
			Message:  &msg,
			Metadata: map[string]interface{}{"session": "abc"},
		}

		// Act
		data, err := json.Marshal(req)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"message":{`)
		assert.Contains(t, string(data), `"session":"abc"`)
	})
}

func TestStreamEvent_Constants(t *testing.T) {
	t.Run("event types have correct values", func(t *testing.T) {
		// Assert
		assert.Equal(t, "message", StreamEventTypeMessage)
		assert.Equal(t, "task", StreamEventTypeTask)
		assert.Equal(t, "error", StreamEventTypeError)
		assert.Equal(t, "done", StreamEventTypeDone)
		assert.Equal(t, "progress", StreamEventTypeProgress)
	})
}

func TestStreamEvent_JSONSerialization(t *testing.T) {
	t.Run("serializes event with message", func(t *testing.T) {
		// Arrange
		msg := NewAssistantMessage("Response")
		event := StreamEvent{
			Type:    StreamEventTypeMessage,
			Message: &msg,
		}

		// Act
		data, err := json.Marshal(event)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(data), `"type":"message"`)
		assert.Contains(t, string(data), `"message":{`)
	})

	t.Run("error field is not serialized", func(t *testing.T) {
		// Arrange
		event := StreamEvent{
			Type:  StreamEventTypeMessage,
			Error: assert.AnError,
		}

		// Act
		data, err := json.Marshal(event)

		// Assert
		require.NoError(t, err)
		// Error has json:"-" so the Error field itself should not appear in output
		// The output should only have "type":"message", not an "error" key
		var parsed map[string]interface{}
		require.NoError(t, json.Unmarshal(data, &parsed))
		_, hasErrorKey := parsed["error"]
		assert.False(t, hasErrorKey, "error field should not be serialized")
	})
}
