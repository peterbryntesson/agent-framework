// Copyright (c) Microsoft. All rights reserved.

package testutil

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
)

// ErrFixtureNotFound is returned when a fixture file does not exist.
var ErrFixtureNotFound = errors.New("fixture file not found")

// ErrInvalidFixture is returned when a fixture file cannot be parsed.
var ErrInvalidFixture = errors.New("invalid fixture format")

// TestDataDir returns the absolute path to the testdata directory.
// It locates the testdata directory relative to this source file,
// ensuring fixtures are found regardless of the working directory.
func TestDataDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		// Fallback to relative path if runtime.Caller fails
		return "testdata"
	}
	// Navigate from testutil/ to parent (go/) then into testdata/
	return filepath.Join(filepath.Dir(filepath.Dir(filename)), "testdata")
}

// LoadFixture reads a fixture file from the testdata directory.
// The path should be relative to the testdata directory (e.g., "messages/user_message.json").
// Returns ErrFixtureNotFound if the file does not exist.
func LoadFixture(path string) ([]byte, error) {
	fullPath := filepath.Join(TestDataDir(), path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrFixtureNotFound
		}
		return nil, err
	}

	return data, nil
}

// LoadFixtureAs reads a fixture file and unmarshals it into the target.
// The path should be relative to the testdata directory.
// Returns ErrFixtureNotFound if the file does not exist.
// Returns ErrInvalidFixture wrapped with the underlying error if parsing fails.
func LoadFixtureAs(path string, target interface{}) error {
	data, err := LoadFixture(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return errors.Join(ErrInvalidFixture, err)
	}

	return nil
}

// MustLoadFixture reads a fixture file and panics if it fails.
// This is useful for test initialization where fixture loading must succeed.
// Use this only when the fixture is known to exist and be valid.
func MustLoadFixture(path string) []byte {
	data, err := LoadFixture(path)
	if err != nil {
		panic("failed to load fixture " + path + ": " + err.Error())
	}
	return data
}

// MustLoadFixtureAs reads a fixture file, unmarshals it, and panics if it fails.
// This is useful for test initialization where fixture loading must succeed.
func MustLoadFixtureAs(path string, target interface{}) {
	if err := LoadFixtureAs(path, target); err != nil {
		panic("failed to load fixture " + path + ": " + err.Error())
	}
}

// JSONEqual compares two JSON byte slices for semantic equality.
// It returns true if both represent the same JSON value,
// ignoring differences in whitespace and key ordering.
func JSONEqual(a, b []byte) bool {
	var objA, objB interface{}

	if err := json.Unmarshal(a, &objA); err != nil {
		return false
	}

	if err := json.Unmarshal(b, &objB); err != nil {
		return false
	}

	// Re-marshal to normalize and compare
	normA, err := json.Marshal(objA)
	if err != nil {
		return false
	}

	normB, err := json.Marshal(objB)
	if err != nil {
		return false
	}

	return bytes.Equal(normA, normB)
}

// FixtureExists checks if a fixture file exists at the given path.
// The path should be relative to the testdata directory.
func FixtureExists(path string) bool {
	fullPath := filepath.Join(TestDataDir(), path)
	_, err := os.Stat(fullPath)
	return err == nil
}

// ListFixtures returns a list of fixture files in the specified subdirectory.
// The dir should be relative to the testdata directory (e.g., "messages").
// Returns an empty slice if the directory does not exist or is empty.
func ListFixtures(dir string) ([]string, error) {
	fullPath := filepath.Join(TestDataDir(), dir)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}
