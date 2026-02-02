// Copyright (c) Microsoft. All rights reserved.

// Package testutil provides helper functions for loading test fixtures
// and common test utilities for the agent framework.
//
// This package is intended for use in tests only and provides convenient
// access to JSON fixtures stored in the testdata directory.
//
// # Loading Fixtures
//
// The package provides functions to load JSON fixtures from testdata subdirectories:
//
//	// Load a single JSON file
//	data, err := testutil.LoadFixture("messages/user_message.json")
//	if err != nil {
//	    t.Fatal(err)
//	}
//
//	// Load and unmarshal into a struct
//	var msg chat.Message
//	err := testutil.LoadFixtureAs("messages/user_message.json", &msg)
//	if err != nil {
//	    t.Fatal(err)
//	}
//
// # Fixture Categories
//
// The testdata directory is organized into subdirectories by type:
//   - messages/ - Message fixtures (user, system, assistant, tool messages)
//   - responses/ - Response fixtures (simple, with metadata, async runs)
//   - sessions/ - Session fixtures (empty, with history, multi-turn)
//
// # Test Helpers
//
// The package also provides helper functions for common test operations:
//
//	// Get the testdata directory path
//	path := testutil.TestDataDir()
//
//	// Check if two JSON values are equivalent
//	ok := testutil.JSONEqual(json1, json2)
package testutil
