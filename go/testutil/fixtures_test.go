// Copyright (c) Microsoft. All rights reserved.

package testutil

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestDataDir_ReturnsValidPath(t *testing.T) {
	// Arrange & Act
	dir := TestDataDir()

	// Assert
	assert.NotEmpty(t, dir)
	assert.True(t, filepath.IsAbs(dir) || strings.Contains(dir, "testdata"))
}

func TestTestDataDir_ContainsExpectedSubdirectories(t *testing.T) {
	// Arrange
	dir := TestDataDir()

	// Assert
	assert.DirExists(t, filepath.Join(dir, "messages"))
	assert.DirExists(t, filepath.Join(dir, "responses"))
	assert.DirExists(t, filepath.Join(dir, "sessions"))
}

func TestLoadFixture_ValidFile(t *testing.T) {
	// Arrange & Act
	data, err := LoadFixture("messages/user_message.json")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "user")
}

func TestLoadFixture_NonExistentFile(t *testing.T) {
	// Arrange & Act
	data, err := LoadFixture("nonexistent/file.json")

	// Assert
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.True(t, errors.Is(err, ErrFixtureNotFound))
}

func TestLoadFixture_AllMessageFixtures(t *testing.T) {
	// Arrange
	fixtures := []string{
		"messages/user_message.json",
		"messages/system_message.json",
		"messages/assistant_message.json",
		"messages/tool_message.json",
		"messages/assistant_with_tool_calls.json",
		"messages/multi_content_message.json",
		"messages/conversation.json",
	}

	// Act & Assert
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			data, err := LoadFixture(fixture)
			require.NoError(t, err, "failed to load %s", fixture)
			assert.NotEmpty(t, data)

			// Verify it's valid JSON
			var parsed interface{}
			err = json.Unmarshal(data, &parsed)
			assert.NoError(t, err, "invalid JSON in %s", fixture)
		})
	}
}

func TestLoadFixture_AllResponseFixtures(t *testing.T) {
	// Arrange
	fixtures := []string{
		"responses/simple_response.json",
		"responses/response_with_metadata.json",
		"responses/tool_call_response.json",
		"responses/truncated_response.json",
		"responses/async_run_in_progress.json",
		"responses/async_run_completed.json",
		"responses/async_run_failed.json",
	}

	// Act & Assert
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			data, err := LoadFixture(fixture)
			require.NoError(t, err, "failed to load %s", fixture)
			assert.NotEmpty(t, data)

			// Verify it's valid JSON
			var parsed interface{}
			err = json.Unmarshal(data, &parsed)
			assert.NoError(t, err, "invalid JSON in %s", fixture)
		})
	}
}

func TestLoadFixture_AllSessionFixtures(t *testing.T) {
	// Arrange
	fixtures := []string{
		"sessions/empty_session.json",
		"sessions/session_with_history.json",
		"sessions/multi_turn_session.json",
		"sessions/session_with_tool_calls.json",
	}

	// Act & Assert
	for _, fixture := range fixtures {
		t.Run(fixture, func(t *testing.T) {
			data, err := LoadFixture(fixture)
			require.NoError(t, err, "failed to load %s", fixture)
			assert.NotEmpty(t, data)

			// Verify it's valid JSON
			var parsed interface{}
			err = json.Unmarshal(data, &parsed)
			assert.NoError(t, err, "invalid JSON in %s", fixture)
		})
	}
}

type testMessage struct {
	Role     string `json:"role"`
	Contents []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"contents,omitempty"`
	Content   string `json:"content,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func TestLoadFixtureAs_ValidStruct(t *testing.T) {
	// Arrange
	var msg testMessage

	// Act
	err := LoadFixtureAs("messages/user_message.json", &msg)

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "user", msg.Role)
	require.NotEmpty(t, msg.Contents)
	assert.Equal(t, "text", msg.Contents[0].Type)
	assert.Equal(t, "Hello, I need help with my project.", msg.Contents[0].Text)
}

func TestLoadFixtureAs_NonExistentFile(t *testing.T) {
	// Arrange
	var msg testMessage

	// Act
	err := LoadFixtureAs("nonexistent/file.json", &msg)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrFixtureNotFound))
}

func TestLoadFixtureAs_InvalidJSON(t *testing.T) {
	// Arrange
	// Create a temporary invalid JSON file
	tempDir := t.TempDir()
	invalidPath := filepath.Join(tempDir, "invalid.json")
	err := os.WriteFile(invalidPath, []byte("{invalid json}"), 0644)
	require.NoError(t, err)

	// Create a mock fixture by temporarily modifying testdata lookup
	// Instead, test with a slice expecting an object
	var msg testMessage

	// Act - try to unmarshal an array into a single object
	err = LoadFixtureAs("messages/conversation.json", &msg)

	// Assert
	assert.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidFixture))
}

func TestMustLoadFixture_ValidFile(t *testing.T) {
	// Arrange & Act & Assert (should not panic)
	data := MustLoadFixture("messages/user_message.json")
	assert.NotEmpty(t, data)
}

func TestMustLoadFixture_NonExistentFile_Panics(t *testing.T) {
	// Arrange
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Contains(t, r.(string), "failed to load fixture")
	}()

	// Act
	MustLoadFixture("nonexistent/file.json")

	// Should not reach here
	t.Fatal("expected panic")
}

func TestMustLoadFixtureAs_ValidFile(t *testing.T) {
	// Arrange
	var msg testMessage

	// Act & Assert (should not panic)
	MustLoadFixtureAs("messages/user_message.json", &msg)
	assert.Equal(t, "user", msg.Role)
}

func TestMustLoadFixtureAs_NonExistentFile_Panics(t *testing.T) {
	// Arrange
	var msg testMessage
	defer func() {
		r := recover()
		assert.NotNil(t, r)
		assert.Contains(t, r.(string), "failed to load fixture")
	}()

	// Act
	MustLoadFixtureAs("nonexistent/file.json", &msg)

	// Should not reach here
	t.Fatal("expected panic")
}

func TestJSONEqual_IdenticalJSON(t *testing.T) {
	// Arrange
	json1 := []byte(`{"name": "test", "value": 42}`)
	json2 := []byte(`{"name": "test", "value": 42}`)

	// Act
	result := JSONEqual(json1, json2)

	// Assert
	assert.True(t, result)
}

func TestJSONEqual_DifferentWhitespace(t *testing.T) {
	// Arrange
	json1 := []byte(`{"name":"test","value":42}`)
	json2 := []byte(`{
		"name": "test",
		"value": 42
	}`)

	// Act
	result := JSONEqual(json1, json2)

	// Assert
	assert.True(t, result)
}

func TestJSONEqual_DifferentKeyOrder(t *testing.T) {
	// Arrange
	json1 := []byte(`{"name": "test", "value": 42}`)
	json2 := []byte(`{"value": 42, "name": "test"}`)

	// Act
	result := JSONEqual(json1, json2)

	// Assert
	assert.True(t, result)
}

func TestJSONEqual_DifferentValues(t *testing.T) {
	// Arrange
	json1 := []byte(`{"name": "test", "value": 42}`)
	json2 := []byte(`{"name": "test", "value": 43}`)

	// Act
	result := JSONEqual(json1, json2)

	// Assert
	assert.False(t, result)
}

func TestJSONEqual_InvalidJSON(t *testing.T) {
	// Arrange
	json1 := []byte(`{"valid": "json"}`)
	json2 := []byte(`{invalid json}`)

	// Act
	result := JSONEqual(json1, json2)

	// Assert
	assert.False(t, result)
}

func TestFixtureExists_ExistingFile(t *testing.T) {
	// Arrange & Act
	exists := FixtureExists("messages/user_message.json")

	// Assert
	assert.True(t, exists)
}

func TestFixtureExists_NonExistentFile(t *testing.T) {
	// Arrange & Act
	exists := FixtureExists("nonexistent/file.json")

	// Assert
	assert.False(t, exists)
}

func TestListFixtures_MessagesDirectory(t *testing.T) {
	// Arrange & Act
	files, err := ListFixtures("messages")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, files)
	assert.Contains(t, files, "user_message.json")
	assert.Contains(t, files, "system_message.json")
	assert.Contains(t, files, "assistant_message.json")
}

func TestListFixtures_ResponsesDirectory(t *testing.T) {
	// Arrange & Act
	files, err := ListFixtures("responses")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, files)
	assert.Contains(t, files, "simple_response.json")
}

func TestListFixtures_SessionsDirectory(t *testing.T) {
	// Arrange & Act
	files, err := ListFixtures("sessions")

	// Assert
	require.NoError(t, err)
	assert.NotEmpty(t, files)
	assert.Contains(t, files, "empty_session.json")
	assert.Contains(t, files, "session_with_history.json")
}

func TestListFixtures_NonExistentDirectory(t *testing.T) {
	// Arrange & Act
	files, err := ListFixtures("nonexistent")

	// Assert
	require.NoError(t, err)
	assert.Empty(t, files)
}

func TestSentinelErrors_AreDistinct(t *testing.T) {
	// Assert
	assert.NotEqual(t, ErrFixtureNotFound, ErrInvalidFixture)
	assert.NotEqual(t, ErrFixtureNotFound.Error(), ErrInvalidFixture.Error())
}

func TestSentinelErrors_HaveDescriptiveMessages(t *testing.T) {
	// Assert
	assert.Contains(t, ErrFixtureNotFound.Error(), "not found")
	assert.Contains(t, ErrInvalidFixture.Error(), "invalid")
}
