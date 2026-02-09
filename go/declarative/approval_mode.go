// Copyright (c) Microsoft. All rights reserved.

package declarative

import (
	"bytes"
	"encoding/json"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnmarshalYAML supports approvalMode provided as a string or object.
func (m *MCPApprovalMode) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		m.Kind = strings.TrimSpace(value.Value)
		return nil
	}

	type alias MCPApprovalMode
	var decoded alias
	if err := value.Decode(&decoded); err != nil {
		return err
	}
	*m = MCPApprovalMode(decoded)
	return nil
}

// UnmarshalJSON supports approvalMode provided as a string or object.
func (m *MCPApprovalMode) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	if data[0] == '"' {
		var kind string
		if err := json.Unmarshal(data, &kind); err != nil {
			return err
		}
		m.Kind = strings.TrimSpace(kind)
		return nil
	}

	type alias MCPApprovalMode
	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*m = MCPApprovalMode(decoded)
	return nil
}
