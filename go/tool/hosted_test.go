// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"testing"
)

func TestHostedWebSearchTool_Interface(t *testing.T) {
	// Arrange
	tool := NewHostedWebSearchTool()

	// Act & Assert
	if tool.Name() != "web_search" {
		t.Errorf("expected name 'web_search', got %q", tool.Name())
	}

	if !tool.IsHosted() {
		t.Error("expected IsHosted() to return true")
	}

	if tool.Parameters() != nil {
		t.Error("expected Parameters() to return nil for hosted tools")
	}
}

func TestHostedWebSearchTool_Invoke(t *testing.T) {
	// Arrange
	tool := NewHostedWebSearchTool()

	// Act
	_, err := tool.Invoke(context.Background(), nil)

	// Assert
	if err == nil {
		t.Error("expected Invoke to return an error for hosted tools")
	}
	if err != errHostedInvocation {
		t.Errorf("expected errHostedInvocation, got %v", err)
	}
}

func TestHostedWebSearchTool_ProviderConfig(t *testing.T) {
	// Arrange
	tool := &HostedWebSearchTool{
		ToolName:          "my_search",
		ToolDescription:   "Custom search tool",
		SearchContextSize: "high",
		UserLocation: &UserLocation{
			Type:        "approximate",
			City:        "Seattle",
			Region:      "Washington",
			Country:     "United States",
			CountryCode: "US",
			Timezone:    "America/Los_Angeles",
		},
		AdditionalProperties: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	// Act
	config := tool.ProviderConfig()

	// Assert
	if config["type"] != "web_search" {
		t.Errorf("expected type 'web_search', got %v", config["type"])
	}
	if config["search_context_size"] != "high" {
		t.Errorf("expected search_context_size 'high', got %v", config["search_context_size"])
	}
	if config["custom_key"] != "custom_value" {
		t.Errorf("expected custom_key 'custom_value', got %v", config["custom_key"])
	}

	location, ok := config["user_location"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user_location to be a map")
	}
	if location["city"] != "Seattle" {
		t.Errorf("expected city 'Seattle', got %v", location["city"])
	}
	if location["country_code"] != "US" {
		t.Errorf("expected country_code 'US', got %v", location["country_code"])
	}
}

func TestHostedCodeInterpreterTool_Interface(t *testing.T) {
	// Arrange
	tool := NewHostedCodeInterpreterTool()

	// Act & Assert
	if tool.Name() != "code_interpreter" {
		t.Errorf("expected name 'code_interpreter', got %q", tool.Name())
	}

	if !tool.IsHosted() {
		t.Error("expected IsHosted() to return true")
	}

	if tool.Parameters() != nil {
		t.Error("expected Parameters() to return nil for hosted tools")
	}
}

func TestHostedCodeInterpreterTool_ProviderConfig(t *testing.T) {
	// Arrange
	tool := &HostedCodeInterpreterTool{
		ToolName:        "my_interpreter",
		ToolDescription: "Custom interpreter",
		Container: &CodeInterpreterContainer{
			Image: "python:3.11",
			EnvVars: map[string]string{
				"DEBUG": "true",
			},
		},
		FileIDs: []string{"file-123", "file-456"},
	}

	// Act
	config := tool.ProviderConfig()

	// Assert
	if config["type"] != "code_interpreter" {
		t.Errorf("expected type 'code_interpreter', got %v", config["type"])
	}

	container, ok := config["container"].(map[string]interface{})
	if !ok {
		t.Fatal("expected container to be a map")
	}
	if container["image"] != "python:3.11" {
		t.Errorf("expected image 'python:3.11', got %v", container["image"])
	}

	fileIDs, ok := config["file_ids"].([]string)
	if !ok {
		t.Fatal("expected file_ids to be a []string")
	}
	if len(fileIDs) != 2 {
		t.Errorf("expected 2 file IDs, got %d", len(fileIDs))
	}
}

func TestHostedFileSearchTool_Interface(t *testing.T) {
	// Arrange
	tool := NewHostedFileSearchTool()

	// Act & Assert
	if tool.Name() != "file_search" {
		t.Errorf("expected name 'file_search', got %q", tool.Name())
	}

	if !tool.IsHosted() {
		t.Error("expected IsHosted() to return true")
	}
}

func TestHostedFileSearchTool_ProviderConfig(t *testing.T) {
	// Arrange
	tool := &HostedFileSearchTool{
		ToolName:       "my_file_search",
		VectorStoreIDs: []string{"vs-123", "vs-456"},
		MaxResults:     10,
		Ranking: &FileSearchRanking{
			Ranker:         "default_2024_08_21",
			ScoreThreshold: 0.75,
		},
	}

	// Act
	config := tool.ProviderConfig()

	// Assert
	if config["type"] != "file_search" {
		t.Errorf("expected type 'file_search', got %v", config["type"])
	}
	if config["max_results"] != 10 {
		t.Errorf("expected max_results 10, got %v", config["max_results"])
	}

	vectorStoreIDs, ok := config["vector_store_ids"].([]string)
	if !ok {
		t.Fatal("expected vector_store_ids to be a []string")
	}
	if len(vectorStoreIDs) != 2 {
		t.Errorf("expected 2 vector store IDs, got %d", len(vectorStoreIDs))
	}

	ranking, ok := config["ranking"].(map[string]interface{})
	if !ok {
		t.Fatal("expected ranking to be a map")
	}
	if ranking["ranker"] != "default_2024_08_21" {
		t.Errorf("expected ranker 'default_2024_08_21', got %v", ranking["ranker"])
	}
	if ranking["score_threshold"] != 0.75 {
		t.Errorf("expected score_threshold 0.75, got %v", ranking["score_threshold"])
	}
}

func TestHostedMCPTool_Interface(t *testing.T) {
	// Arrange
	tool := NewHostedMCPTool("my_mcp", "https://example.com/mcp")

	// Act & Assert
	if tool.Name() != "my_mcp" {
		t.Errorf("expected name 'my_mcp', got %q", tool.Name())
	}

	if !tool.IsHosted() {
		t.Error("expected IsHosted() to return true")
	}
}

func TestHostedMCPTool_ProviderConfig_SimpleApproval(t *testing.T) {
	// Arrange
	tool := &HostedMCPTool{
		ToolName:        "my_mcp",
		ToolDescription: "MCP server integration",
		ServerURL:       "https://example.com/mcp",
		ServerLabel:     "Example MCP",
		AllowedTools:    []string{"tool1", "tool2"},
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
		RequireApproval: MCPApprovalAlways,
	}

	// Act
	config := tool.ProviderConfig()

	// Assert
	if config["type"] != "mcp" {
		t.Errorf("expected type 'mcp', got %v", config["type"])
	}
	if config["server_url"] != "https://example.com/mcp" {
		t.Errorf("expected server_url 'https://example.com/mcp', got %v", config["server_url"])
	}
	if config["server_label"] != "Example MCP" {
		t.Errorf("expected server_label 'Example MCP', got %v", config["server_label"])
	}
	if config["require_approval"] != "always" {
		t.Errorf("expected require_approval 'always', got %v", config["require_approval"])
	}

	allowedTools, ok := config["allowed_tools"].([]string)
	if !ok {
		t.Fatal("expected allowed_tools to be a []string")
	}
	if len(allowedTools) != 2 {
		t.Errorf("expected 2 allowed tools, got %d", len(allowedTools))
	}

	headers, ok := config["headers"].(map[string]string)
	if !ok {
		t.Fatal("expected headers to be a map[string]string")
	}
	if headers["Authorization"] != "Bearer token123" {
		t.Errorf("expected Authorization header, got %v", headers["Authorization"])
	}
}

func TestHostedMCPTool_ProviderConfig_SpecificApproval(t *testing.T) {
	// Arrange
	tool := &HostedMCPTool{
		ToolName:  "my_mcp",
		ServerURL: "https://example.com/mcp",
		SpecificApproval: &MCPSpecificApproval{
			AlwaysRequireApproval: []string{"dangerous_tool"},
			NeverRequireApproval:  []string{"safe_tool", "readonly_tool"},
		},
	}

	// Act
	config := tool.ProviderConfig()

	// Assert
	approval, ok := config["require_approval"].(map[string]interface{})
	if !ok {
		t.Fatal("expected require_approval to be a map for specific approval")
	}

	alwaysApproval, ok := approval["always_require_approval"].([]string)
	if !ok {
		t.Fatal("expected always_require_approval to be a []string")
	}
	if len(alwaysApproval) != 1 || alwaysApproval[0] != "dangerous_tool" {
		t.Errorf("unexpected always_require_approval: %v", alwaysApproval)
	}

	neverApproval, ok := approval["never_require_approval"].([]string)
	if !ok {
		t.Fatal("expected never_require_approval to be a []string")
	}
	if len(neverApproval) != 2 {
		t.Errorf("expected 2 never_require_approval tools, got %d", len(neverApproval))
	}
}

func TestHostedImageGenerationTool_Interface(t *testing.T) {
	// Arrange
	tool := NewHostedImageGenerationTool()

	// Act & Assert
	if tool.Name() != "image_generation" {
		t.Errorf("expected name 'image_generation', got %q", tool.Name())
	}

	if !tool.IsHosted() {
		t.Error("expected IsHosted() to return true")
	}
}

func TestHostedImageGenerationTool_ProviderConfig(t *testing.T) {
	// Arrange
	tool := &HostedImageGenerationTool{
		ToolName:          "my_image_gen",
		ToolDescription:   "Image generator",
		Quality:           ImageQualityHigh,
		OutputFormat:      ImageFormatPNG,
		OutputCompression: 85,
		Background:        ImageBackgroundTransparent,
		Size:              "1024x1024",
		PartialImages:     true,
	}

	// Act
	config := tool.ProviderConfig()

	// Assert
	if config["type"] != "image_generation" {
		t.Errorf("expected type 'image_generation', got %v", config["type"])
	}
	if config["quality"] != "high" {
		t.Errorf("expected quality 'high', got %v", config["quality"])
	}
	if config["output_format"] != "png" {
		t.Errorf("expected output_format 'png', got %v", config["output_format"])
	}
	if config["output_compression"] != 85 {
		t.Errorf("expected output_compression 85, got %v", config["output_compression"])
	}
	if config["background"] != "transparent" {
		t.Errorf("expected background 'transparent', got %v", config["background"])
	}
	if config["size"] != "1024x1024" {
		t.Errorf("expected size '1024x1024', got %v", config["size"])
	}
	if config["partial_images"] != true {
		t.Errorf("expected partial_images true, got %v", config["partial_images"])
	}
}

func TestHostedTools_DefaultNames(t *testing.T) {
	tests := []struct {
		name     string
		tool     HostedTool
		expected string
	}{
		{"WebSearch empty name", &HostedWebSearchTool{}, "web_search"},
		{"CodeInterpreter empty name", &HostedCodeInterpreterTool{}, "code_interpreter"},
		{"FileSearch empty name", &HostedFileSearchTool{}, "file_search"},
		{"ImageGeneration empty name", &HostedImageGenerationTool{}, "image_generation"},
		{"WebSearch custom name", &HostedWebSearchTool{ToolName: "custom"}, "custom"},
		{"CodeInterpreter custom name", &HostedCodeInterpreterTool{ToolName: "custom"}, "custom"},
		{"FileSearch custom name", &HostedFileSearchTool{ToolName: "custom"}, "custom"},
		{"ImageGeneration custom name", &HostedImageGenerationTool{ToolName: "custom"}, "custom"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tool.Name(); got != tc.expected {
				t.Errorf("expected name %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestHostedTools_AllReturnHostedTrue(t *testing.T) {
	tools := []HostedTool{
		NewHostedWebSearchTool(),
		NewHostedCodeInterpreterTool(),
		NewHostedFileSearchTool(),
		NewHostedMCPTool("test", "https://example.com"),
		NewHostedImageGenerationTool(),
	}

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			if !tool.IsHosted() {
				t.Errorf("expected IsHosted() to return true for %s", tool.Name())
			}
		})
	}
}

func TestHostedTools_InvokeReturnsError(t *testing.T) {
	tools := []HostedTool{
		NewHostedWebSearchTool(),
		NewHostedCodeInterpreterTool(),
		NewHostedFileSearchTool(),
		NewHostedMCPTool("test", "https://example.com"),
		NewHostedImageGenerationTool(),
	}

	ctx := context.Background()
	args := json.RawMessage(`{}`)

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			_, err := tool.Invoke(ctx, args)
			if err != errHostedInvocation {
				t.Errorf("expected errHostedInvocation for %s, got %v", tool.Name(), err)
			}
		})
	}
}

func TestHostedTools_ProviderConfigIncludesType(t *testing.T) {
	tests := []struct {
		name         string
		tool         HostedTool
		expectedType string
	}{
		{"WebSearch", NewHostedWebSearchTool(), "web_search"},
		{"CodeInterpreter", NewHostedCodeInterpreterTool(), "code_interpreter"},
		{"FileSearch", NewHostedFileSearchTool(), "file_search"},
		{"MCP", NewHostedMCPTool("test", "https://example.com"), "mcp"},
		{"ImageGeneration", NewHostedImageGenerationTool(), "image_generation"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config := tc.tool.ProviderConfig()
			if config["type"] != tc.expectedType {
				t.Errorf("expected type %q, got %v", tc.expectedType, config["type"])
			}
		})
	}
}

func TestHostedWebSearchTool_MinimalConfig(t *testing.T) {
	// Arrange
	tool := NewHostedWebSearchTool()

	// Act
	config := tool.ProviderConfig()

	// Assert
	if len(config) != 1 {
		t.Errorf("expected minimal config with only 'type', got %d keys", len(config))
	}
	if config["type"] != "web_search" {
		t.Errorf("expected type 'web_search', got %v", config["type"])
	}
}

func TestHostedCodeInterpreterTool_MinimalConfig(t *testing.T) {
	// Arrange
	tool := NewHostedCodeInterpreterTool()

	// Act
	config := tool.ProviderConfig()

	// Assert
	if len(config) != 1 {
		t.Errorf("expected minimal config with only 'type', got %d keys", len(config))
	}
}

func TestHostedFileSearchTool_MinimalConfig(t *testing.T) {
	// Arrange
	tool := NewHostedFileSearchTool()

	// Act
	config := tool.ProviderConfig()

	// Assert
	if len(config) != 1 {
		t.Errorf("expected minimal config with only 'type', got %d keys", len(config))
	}
}

func TestHostedMCPTool_MinimalConfig(t *testing.T) {
	// Arrange
	tool := NewHostedMCPTool("test", "https://example.com")

	// Act
	config := tool.ProviderConfig()

	// Assert
	if config["type"] != "mcp" {
		t.Errorf("expected type 'mcp', got %v", config["type"])
	}
	if config["server_url"] != "https://example.com" {
		t.Errorf("expected server_url 'https://example.com', got %v", config["server_url"])
	}
}

func TestHostedImageGenerationTool_MinimalConfig(t *testing.T) {
	// Arrange
	tool := NewHostedImageGenerationTool()

	// Act
	config := tool.ProviderConfig()

	// Assert
	if len(config) != 1 {
		t.Errorf("expected minimal config with only 'type', got %d keys", len(config))
	}
}
