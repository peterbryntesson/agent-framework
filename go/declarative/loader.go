// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadFromFile loads a PromptAgent from a YAML file.
// The file path should point to a valid YAML agent definition.
func LoadFromFile(path string) (*PromptAgent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer f.Close()
	return LoadFromReader(f)
}

// LoadFromReader loads a PromptAgent from a reader.
// The reader should provide valid YAML content.
func LoadFromReader(r io.Reader) (*PromptAgent, error) {
	var agent PromptAgent
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&agent); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	if err := validate(&agent); err != nil {
		return nil, err
	}

	return &agent, nil
}

// LoadFromString loads a PromptAgent from a YAML string.
func LoadFromString(content string) (*PromptAgent, error) {
	return LoadFromReader(strings.NewReader(content))
}

// LoadFromBytes loads a PromptAgent from YAML bytes.
func LoadFromBytes(data []byte) (*PromptAgent, error) {
	return LoadFromString(string(data))
}
