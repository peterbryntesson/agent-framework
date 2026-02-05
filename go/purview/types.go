// Copyright (c) Microsoft. All rights reserved.

package purview

import (
	"context"
	"time"
)

// Settings configures Purview content policy evaluation.
type Settings struct {
	// TenantID is the Azure AD tenant ID.
	TenantID string

	// ContentPolicy is the name of the content policy to evaluate.
	ContentPolicy string

	// Categories specifies which content categories to check.
	// If empty, all categories defined in the policy are checked.
	Categories []string

	// BlockOnViolation determines whether to block content that violates policies.
	// Default is true.
	BlockOnViolation bool

	// CacheScopes enables caching of scope definitions.
	// Default is false.
	CacheScopes bool
}

// PolicyResult contains the result of content policy evaluation.
type PolicyResult struct {
	// IsViolation indicates whether the content violates the policy.
	IsViolation bool

	// PolicyName is the name of the evaluated policy.
	PolicyName string

	// Violations contains details of detected violations.
	Violations []Violation

	// Scores contains confidence scores for each category.
	Scores map[string]float64

	// ProcessedAt is when the evaluation was performed.
	ProcessedAt time.Time
}

// Violation represents a single policy violation.
type Violation struct {
	// Category is the content category that was violated.
	Category string

	// Severity indicates the severity level (low, medium, high, critical).
	Severity string

	// Confidence is the confidence score (0-1) of the violation detection.
	Confidence float64

	// Description provides human-readable details about the violation.
	Description string

	// Location identifies where in the content the violation was detected.
	Location *ContentLocation
}

// ContentLocation specifies a location within content.
type ContentLocation struct {
	// StartOffset is the character offset where the violation begins.
	StartOffset int

	// EndOffset is the character offset where the violation ends.
	EndOffset int

	// Text is the violating text segment.
	Text string
}

// ViolationHandler is called when a policy violation is detected.
// Return nil to allow content despite violations, or an error to block.
type ViolationHandler func(ctx context.Context, result *PolicyResult) error

// ScopeCache caches scope definitions.
type ScopeCache interface {
	// Get retrieves a cached scope definition.
	Get(ctx context.Context, key string) ([]byte, string, bool)

	// Set stores a scope definition with the given ETag.
	Set(ctx context.Context, key string, data []byte, etag string, ttl time.Duration)
}

// EvaluationRequest is the request body for content evaluation.
type EvaluationRequest struct {
	Content    string   `json:"content"`
	Categories []string `json:"categories,omitempty"`
	PolicyName string   `json:"policyName"`
}

// EvaluationResponse is the response from content evaluation.
type EvaluationResponse struct {
	IsViolation bool                `json:"isViolation"`
	PolicyName  string              `json:"policyName"`
	Violations  []ViolationResponse `json:"violations,omitempty"`
	Scores      map[string]float64  `json:"scores,omitempty"`
}

// ViolationResponse is a violation in the API response.
type ViolationResponse struct {
	Category    string  `json:"category"`
	Severity    string  `json:"severity"`
	Confidence  float64 `json:"confidence"`
	Description string  `json:"description,omitempty"`
	StartOffset *int    `json:"startOffset,omitempty"`
	EndOffset   *int    `json:"endOffset,omitempty"`
	Text        string  `json:"text,omitempty"`
}

// ScopeDefinition represents a cached scope definition.
type ScopeDefinition struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Categories []string `json:"categories"`
	ETag       string   `json:"etag"`
}

// ContentType specifies the type of content being evaluated.
type ContentType string

const (
	// ContentTypeText represents plain text content.
	ContentTypeText ContentType = "text"

	// ContentTypeHTML represents HTML content.
	ContentTypeHTML ContentType = "html"

	// ContentTypeMarkdown represents Markdown content.
	ContentTypeMarkdown ContentType = "markdown"
)

// PolicyViolationError is returned when content violates a policy.
type PolicyViolationError struct {
	Result *PolicyResult
}

// Error implements error.
func (e *PolicyViolationError) Error() string {
	if len(e.Result.Violations) > 0 {
		return "content policy violation: " + e.Result.Violations[0].Category
	}
	return "content policy violation"
}
