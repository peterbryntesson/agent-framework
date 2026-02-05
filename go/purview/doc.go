// Copyright (c) Microsoft. All rights reserved.

/*
Package purview provides Microsoft Purview content policy integration for AI agents.

The purview package enables agents to evaluate content against Microsoft Purview
content policies before processing requests or returning responses. This helps
organizations enforce compliance requirements and content governance policies
for AI-generated content.

# Overview

Microsoft Purview provides content classification, sensitivity labeling, and
policy enforcement capabilities. This package integrates those capabilities
into the agent framework through middleware that can:

  - Evaluate input content against content policies
  - Evaluate output content before returning to users
  - Block or modify content that violates policies
  - Log policy violations for audit purposes

# Quick Start

To add Purview content policy evaluation to an agent:

	// Create credential (using Azure Identity)
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
	    log.Fatal(err)
	}

	// Create Purview middleware
	middleware := purview.NewMiddleware(cred, purview.Settings{
	    TenantID:      "your-tenant-id",
	    ContentPolicy: "AI-Content-Policy",
	})

	// Apply to agent using the agent builder
	myAgent := chatagent.New("assistant", chatClient,
	    chatagent.WithMiddleware(middleware.Apply()),
	)

# Settings

The Settings struct configures Purview integration:

	settings := purview.Settings{
	    TenantID:       "your-tenant-id",       // Required: Azure AD tenant
	    ContentPolicy:  "AI-Content-Policy",    // Policy name to evaluate
	    Categories:     []string{"PII", "NSFW"}, // Categories to check
	    BlockOnViolation: true,                  // Block violating content
	    CacheScopes:    true,                    // Cache scope definitions
	}

# Response Handling

When content violates a policy, the middleware can:

  - Return an error (default with BlockOnViolation)
  - Allow content with policy violation metadata
  - Invoke a custom violation handler

Custom violation handling:

	middleware := purview.NewMiddleware(cred, settings,
	    purview.WithViolationHandler(func(ctx context.Context, result *purview.PolicyResult) error {
	        log.Printf("Policy violation: %s", result.PolicyName)
	        return nil // Allow content despite violation
	    }),
	)

# Caching

Scope definitions can be cached to reduce API calls:

	middleware := purview.NewMiddleware(cred, settings,
	    purview.WithCache(myCache),
	    purview.WithCacheTTL(1 * time.Hour),
	)

The cache uses ETags for efficient revalidation.
*/
package purview
