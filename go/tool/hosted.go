// Copyright (c) Microsoft. All rights reserved.

package tool

import (
	"context"
	"encoding/json"
	"errors"
)

// errHostedInvocation is returned when attempting to invoke a hosted tool locally.
var errHostedInvocation = errors.New("hosted tools are invoked by the provider, not locally")

// UserLocation represents approximate user location for context-aware search.
// This information can be provided to hosted web search tools to improve
// relevance of search results based on geographic context.
type UserLocation struct {
	// Type describes the precision of the location (e.g., "approximate").
	Type string `json:"type,omitempty"`

	// City is the city name.
	City string `json:"city,omitempty"`

	// Region is the state, province, or region name.
	Region string `json:"region,omitempty"`

	// Country is the full country name.
	Country string `json:"country,omitempty"`

	// CountryCode is the ISO country code (e.g., "US", "GB").
	CountryCode string `json:"country_code,omitempty"`

	// Timezone is the IANA timezone identifier (e.g., "America/Los_Angeles").
	Timezone string `json:"timezone,omitempty"`
}

// HostedWebSearchTool represents a provider-hosted web search capability.
// This tool enables AI models to search the web for real-time information
// during conversations. The search is executed by the provider's infrastructure.
//
// Aligns with Python HostedWebSearchTool and .NET ResponseWebSearchToolDefinition.
type HostedWebSearchTool struct {
	// ToolName is the identifier for this tool. Defaults to "web_search".
	ToolName string

	// ToolDescription describes what this tool does.
	ToolDescription string

	// SearchContextSize controls the amount of context to retrieve.
	// Valid values: "low", "medium", "high". Defaults to provider's default.
	SearchContextSize string

	// UserLocation provides geographic context for search relevance.
	UserLocation *UserLocation

	// AdditionalProperties contains provider-specific configuration.
	AdditionalProperties map[string]interface{}
}

// NewHostedWebSearchTool creates a new web search tool with default settings.
func NewHostedWebSearchTool() *HostedWebSearchTool {
	return &HostedWebSearchTool{
		ToolName: "web_search",
	}
}

// Name returns the tool identifier.
func (t *HostedWebSearchTool) Name() string {
	if t.ToolName == "" {
		return "web_search"
	}
	return t.ToolName
}

// Description returns the tool description.
func (t *HostedWebSearchTool) Description() string {
	return t.ToolDescription
}

// Parameters returns nil as hosted tools don't have user-defined parameters.
func (t *HostedWebSearchTool) Parameters() json.RawMessage {
	return nil
}

// Invoke returns an error because hosted tools are executed by the provider.
func (t *HostedWebSearchTool) Invoke(ctx context.Context, arguments json.RawMessage) (Result, error) {
	return Result{}, errHostedInvocation
}

// IsHosted returns true, indicating this tool runs on the provider.
func (t *HostedWebSearchTool) IsHosted() bool {
	return true
}

// ProviderConfig returns the provider-specific configuration for web search.
func (t *HostedWebSearchTool) ProviderConfig() map[string]interface{} {
	config := map[string]interface{}{
		"type": "web_search",
	}

	if t.SearchContextSize != "" {
		config["search_context_size"] = t.SearchContextSize
	}

	if t.UserLocation != nil {
		location := map[string]interface{}{
			"type": t.UserLocation.Type,
		}
		if t.UserLocation.City != "" {
			location["city"] = t.UserLocation.City
		}
		if t.UserLocation.Region != "" {
			location["region"] = t.UserLocation.Region
		}
		if t.UserLocation.Country != "" {
			location["country"] = t.UserLocation.Country
		}
		if t.UserLocation.CountryCode != "" {
			location["country_code"] = t.UserLocation.CountryCode
		}
		if t.UserLocation.Timezone != "" {
			location["timezone"] = t.UserLocation.Timezone
		}
		config["user_location"] = location
	}

	// Merge additional properties
	for k, v := range t.AdditionalProperties {
		config[k] = v
	}

	return config
}

// CodeInterpreterContainer represents a container configuration for code execution.
type CodeInterpreterContainer struct {
	// Image is the container image to use (e.g., "python:3.11").
	Image string `json:"image,omitempty"`

	// EnvVars are environment variables to set in the container.
	EnvVars map[string]string `json:"env_vars,omitempty"`
}

// HostedCodeInterpreterTool represents a provider-hosted code execution environment.
// This tool allows AI models to write and execute code during conversations.
// The execution happens in a sandboxed environment on the provider's infrastructure.
//
// Aligns with Python HostedCodeInterpreterTool.
type HostedCodeInterpreterTool struct {
	// ToolName is the identifier for this tool. Defaults to "code_interpreter".
	ToolName string

	// ToolDescription describes what this tool does.
	ToolDescription string

	// Container specifies custom container configuration for execution.
	Container *CodeInterpreterContainer

	// FileIDs are IDs of files available to the code interpreter.
	FileIDs []string

	// AdditionalProperties contains provider-specific configuration.
	AdditionalProperties map[string]interface{}
}

// NewHostedCodeInterpreterTool creates a new code interpreter tool with default settings.
func NewHostedCodeInterpreterTool() *HostedCodeInterpreterTool {
	return &HostedCodeInterpreterTool{
		ToolName: "code_interpreter",
	}
}

// Name returns the tool identifier.
func (t *HostedCodeInterpreterTool) Name() string {
	if t.ToolName == "" {
		return "code_interpreter"
	}
	return t.ToolName
}

// Description returns the tool description.
func (t *HostedCodeInterpreterTool) Description() string {
	return t.ToolDescription
}

// Parameters returns nil as hosted tools don't have user-defined parameters.
func (t *HostedCodeInterpreterTool) Parameters() json.RawMessage {
	return nil
}

// Invoke returns an error because hosted tools are executed by the provider.
func (t *HostedCodeInterpreterTool) Invoke(ctx context.Context, arguments json.RawMessage) (Result, error) {
	return Result{}, errHostedInvocation
}

// IsHosted returns true, indicating this tool runs on the provider.
func (t *HostedCodeInterpreterTool) IsHosted() bool {
	return true
}

// ProviderConfig returns the provider-specific configuration for code interpreter.
func (t *HostedCodeInterpreterTool) ProviderConfig() map[string]interface{} {
	config := map[string]interface{}{
		"type": "code_interpreter",
	}

	if t.Container != nil {
		container := map[string]interface{}{}
		if t.Container.Image != "" {
			container["image"] = t.Container.Image
		}
		if len(t.Container.EnvVars) > 0 {
			container["env_vars"] = t.Container.EnvVars
		}
		config["container"] = container
	}

	if len(t.FileIDs) > 0 {
		config["file_ids"] = t.FileIDs
	}

	// Merge additional properties
	for k, v := range t.AdditionalProperties {
		config[k] = v
	}

	return config
}

// FileSearchRanking specifies ranking configuration for file search results.
type FileSearchRanking struct {
	// Ranker is the ranking algorithm to use (e.g., "default_2024_08_21").
	Ranker string `json:"ranker,omitempty"`

	// ScoreThreshold is the minimum relevance score for results (0.0 to 1.0).
	ScoreThreshold float64 `json:"score_threshold,omitempty"`
}

// HostedFileSearchTool represents a provider-hosted vector search capability.
// This tool enables AI models to search through uploaded files using
// semantic/vector search. Results are retrieved from the provider's
// vector store infrastructure.
//
// Aligns with Python HostedFileSearchTool.
type HostedFileSearchTool struct {
	// ToolName is the identifier for this tool. Defaults to "file_search".
	ToolName string

	// ToolDescription describes what this tool does.
	ToolDescription string

	// VectorStoreIDs are IDs of vector stores to search.
	VectorStoreIDs []string

	// MaxResults limits the number of search results returned.
	MaxResults int

	// Ranking specifies the ranking configuration for results.
	Ranking *FileSearchRanking

	// AdditionalProperties contains provider-specific configuration.
	AdditionalProperties map[string]interface{}
}

// NewHostedFileSearchTool creates a new file search tool with default settings.
func NewHostedFileSearchTool() *HostedFileSearchTool {
	return &HostedFileSearchTool{
		ToolName: "file_search",
	}
}

// Name returns the tool identifier.
func (t *HostedFileSearchTool) Name() string {
	if t.ToolName == "" {
		return "file_search"
	}
	return t.ToolName
}

// Description returns the tool description.
func (t *HostedFileSearchTool) Description() string {
	return t.ToolDescription
}

// Parameters returns nil as hosted tools don't have user-defined parameters.
func (t *HostedFileSearchTool) Parameters() json.RawMessage {
	return nil
}

// Invoke returns an error because hosted tools are executed by the provider.
func (t *HostedFileSearchTool) Invoke(ctx context.Context, arguments json.RawMessage) (Result, error) {
	return Result{}, errHostedInvocation
}

// IsHosted returns true, indicating this tool runs on the provider.
func (t *HostedFileSearchTool) IsHosted() bool {
	return true
}

// ProviderConfig returns the provider-specific configuration for file search.
func (t *HostedFileSearchTool) ProviderConfig() map[string]interface{} {
	config := map[string]interface{}{
		"type": "file_search",
	}

	if len(t.VectorStoreIDs) > 0 {
		config["vector_store_ids"] = t.VectorStoreIDs
	}

	if t.MaxResults > 0 {
		config["max_results"] = t.MaxResults
	}

	if t.Ranking != nil {
		ranking := map[string]interface{}{}
		if t.Ranking.Ranker != "" {
			ranking["ranker"] = t.Ranking.Ranker
		}
		if t.Ranking.ScoreThreshold > 0 {
			ranking["score_threshold"] = t.Ranking.ScoreThreshold
		}
		config["ranking"] = ranking
	}

	// Merge additional properties
	for k, v := range t.AdditionalProperties {
		config[k] = v
	}

	return config
}

// MCPApprovalMode specifies when user approval is required for MCP tool invocation.
type MCPApprovalMode string

const (
	// MCPApprovalNever means the MCP tool never requires user approval.
	MCPApprovalNever MCPApprovalMode = "never"

	// MCPApprovalAlways means the MCP tool always requires user approval.
	MCPApprovalAlways MCPApprovalMode = "always"
)

// MCPSpecificApproval provides fine-grained approval control for individual tools.
type MCPSpecificApproval struct {
	// AlwaysRequireApproval lists tools that always require user approval.
	AlwaysRequireApproval []string

	// NeverRequireApproval lists tools that never require user approval.
	NeverRequireApproval []string
}

// HostedMCPTool represents a Model Context Protocol server integration.
// MCP allows connecting to external tool servers that provide dynamic
// tool capabilities. The MCP server is accessed by the provider.
//
// Aligns with Python HostedMCPTool.
type HostedMCPTool struct {
	// ToolName is the identifier for this tool.
	ToolName string

	// ToolDescription describes what this tool does.
	ToolDescription string

	// ServerURL is the URL of the MCP server.
	ServerURL string

	// ServerLabel is a human-readable label for the server.
	ServerLabel string

	// AllowedTools restricts which tools from the MCP server can be used.
	// If nil or empty, all tools from the server are available.
	AllowedTools []string

	// Headers are HTTP headers to include in requests to the MCP server.
	Headers map[string]string

	// RequireApproval specifies approval mode as a simple string ("never" or "always").
	RequireApproval MCPApprovalMode

	// SpecificApproval provides fine-grained approval control per tool.
	// If set, RequireApproval is ignored.
	SpecificApproval *MCPSpecificApproval

	// AdditionalProperties contains provider-specific configuration.
	AdditionalProperties map[string]interface{}
}

// NewHostedMCPTool creates a new MCP tool with the given name and server URL.
func NewHostedMCPTool(name, serverURL string) *HostedMCPTool {
	return &HostedMCPTool{
		ToolName:  name,
		ServerURL: serverURL,
	}
}

// Name returns the tool identifier.
func (t *HostedMCPTool) Name() string {
	return t.ToolName
}

// Description returns the tool description.
func (t *HostedMCPTool) Description() string {
	return t.ToolDescription
}

// Parameters returns nil as hosted tools don't have user-defined parameters.
func (t *HostedMCPTool) Parameters() json.RawMessage {
	return nil
}

// Invoke returns an error because hosted tools are executed by the provider.
func (t *HostedMCPTool) Invoke(ctx context.Context, arguments json.RawMessage) (Result, error) {
	return Result{}, errHostedInvocation
}

// IsHosted returns true, indicating this tool runs on the provider.
func (t *HostedMCPTool) IsHosted() bool {
	return true
}

// ProviderConfig returns the provider-specific configuration for MCP tool.
func (t *HostedMCPTool) ProviderConfig() map[string]interface{} {
	config := map[string]interface{}{
		"type":       "mcp",
		"server_url": t.ServerURL,
	}

	if t.ServerLabel != "" {
		config["server_label"] = t.ServerLabel
	}

	if len(t.AllowedTools) > 0 {
		config["allowed_tools"] = t.AllowedTools
	}

	if len(t.Headers) > 0 {
		config["headers"] = t.Headers
	}

	// Handle approval modes
	if t.SpecificApproval != nil {
		approval := map[string]interface{}{}
		if len(t.SpecificApproval.AlwaysRequireApproval) > 0 {
			approval["always_require_approval"] = t.SpecificApproval.AlwaysRequireApproval
		}
		if len(t.SpecificApproval.NeverRequireApproval) > 0 {
			approval["never_require_approval"] = t.SpecificApproval.NeverRequireApproval
		}
		config["require_approval"] = approval
	} else if t.RequireApproval != "" {
		config["require_approval"] = string(t.RequireApproval)
	}

	// Merge additional properties
	for k, v := range t.AdditionalProperties {
		config[k] = v
	}

	return config
}

// ImageQuality specifies the quality level for generated images.
type ImageQuality string

const (
	// ImageQualityLow generates lower quality, faster images.
	ImageQualityLow ImageQuality = "low"

	// ImageQualityMedium generates medium quality images.
	ImageQualityMedium ImageQuality = "medium"

	// ImageQualityHigh generates higher quality, slower images.
	ImageQualityHigh ImageQuality = "high"

	// ImageQualityAuto lets the provider decide quality based on the prompt.
	ImageQualityAuto ImageQuality = "auto"
)

// ImageOutputFormat specifies the format for generated images.
type ImageOutputFormat string

const (
	// ImageFormatPNG outputs images in PNG format.
	ImageFormatPNG ImageOutputFormat = "png"

	// ImageFormatJPEG outputs images in JPEG format.
	ImageFormatJPEG ImageOutputFormat = "jpeg"

	// ImageFormatWebP outputs images in WebP format.
	ImageFormatWebP ImageOutputFormat = "webp"
)

// ImageBackground specifies the background mode for generated images.
type ImageBackground string

const (
	// ImageBackgroundTransparent generates images with transparent backgrounds.
	ImageBackgroundTransparent ImageBackground = "transparent"

	// ImageBackgroundOpaque generates images with opaque backgrounds.
	ImageBackgroundOpaque ImageBackground = "opaque"

	// ImageBackgroundAuto lets the provider decide based on the format and prompt.
	ImageBackgroundAuto ImageBackground = "auto"
)

// HostedImageGenerationTool represents a provider-hosted image generation capability.
// This tool enables AI models to generate images based on text descriptions.
// Image generation is performed on the provider's infrastructure.
//
// Aligns with Python HostedImageGenerationTool.
type HostedImageGenerationTool struct {
	// ToolName is the identifier for this tool. Defaults to "image_generation".
	ToolName string

	// ToolDescription describes what this tool does.
	ToolDescription string

	// Quality specifies the quality level for generated images.
	Quality ImageQuality

	// OutputFormat specifies the image output format.
	OutputFormat ImageOutputFormat

	// OutputCompression specifies compression level (0-100) for JPEG/WebP.
	OutputCompression int

	// Background specifies the background mode.
	Background ImageBackground

	// Size specifies the image dimensions (e.g., "1024x1024", "1792x1024").
	Size string

	// PartialImages enables streaming partial image results during generation.
	PartialImages bool

	// AdditionalProperties contains provider-specific configuration.
	AdditionalProperties map[string]interface{}
}

// NewHostedImageGenerationTool creates a new image generation tool with default settings.
func NewHostedImageGenerationTool() *HostedImageGenerationTool {
	return &HostedImageGenerationTool{
		ToolName: "image_generation",
	}
}

// Name returns the tool identifier.
func (t *HostedImageGenerationTool) Name() string {
	if t.ToolName == "" {
		return "image_generation"
	}
	return t.ToolName
}

// Description returns the tool description.
func (t *HostedImageGenerationTool) Description() string {
	return t.ToolDescription
}

// Parameters returns nil as hosted tools don't have user-defined parameters.
func (t *HostedImageGenerationTool) Parameters() json.RawMessage {
	return nil
}

// Invoke returns an error because hosted tools are executed by the provider.
func (t *HostedImageGenerationTool) Invoke(ctx context.Context, arguments json.RawMessage) (Result, error) {
	return Result{}, errHostedInvocation
}

// IsHosted returns true, indicating this tool runs on the provider.
func (t *HostedImageGenerationTool) IsHosted() bool {
	return true
}

// ProviderConfig returns the provider-specific configuration for image generation.
func (t *HostedImageGenerationTool) ProviderConfig() map[string]interface{} {
	config := map[string]interface{}{
		"type": "image_generation",
	}

	if t.Quality != "" {
		config["quality"] = string(t.Quality)
	}

	if t.OutputFormat != "" {
		config["output_format"] = string(t.OutputFormat)
	}

	if t.OutputCompression > 0 {
		config["output_compression"] = t.OutputCompression
	}

	if t.Background != "" {
		config["background"] = string(t.Background)
	}

	if t.Size != "" {
		config["size"] = t.Size
	}

	if t.PartialImages {
		config["partial_images"] = true
	}

	// Merge additional properties
	for k, v := range t.AdditionalProperties {
		config[k] = v
	}

	return config
}

// Compile-time interface assertions
var (
	_ HostedTool = (*HostedWebSearchTool)(nil)
	_ HostedTool = (*HostedCodeInterpreterTool)(nil)
	_ HostedTool = (*HostedFileSearchTool)(nil)
	_ HostedTool = (*HostedMCPTool)(nil)
	_ HostedTool = (*HostedImageGenerationTool)(nil)
)
