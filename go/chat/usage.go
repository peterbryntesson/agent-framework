// Copyright (c) Microsoft. All rights reserved.

package chat

// UsageDetails contains token usage information from a chat completion.
// This is used for billing, monitoring, and optimization purposes.
type UsageDetails struct {
	// InputTokens is the number of tokens in the input messages.
	InputTokens int

	// OutputTokens is the number of tokens in the generated response.
	OutputTokens int

	// TotalTokens is the total number of tokens used (input + output).
	TotalTokens int

	// CachedTokens is the number of tokens served from cache (optional).
	// This may be zero if the provider doesn't support or report caching.
	CachedTokens int

	// ReasoningTokens is the number of tokens used for reasoning (optional).
	// This is reported by models that use chain-of-thought reasoning.
	ReasoningTokens int
}
