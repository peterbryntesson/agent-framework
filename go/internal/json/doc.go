// Copyright (c) Microsoft. All rights reserved.

/*
Package json provides utilities for efficient JSON marshaling and unmarshaling.

This internal package offers helper functions for common JSON operations used
throughout the agent framework, with proper handling of nil and empty values.

# Basic Usage

Use MarshalToRawMessage to convert Go values to json.RawMessage:

	type Config struct {
		Name string `json:"name"`
	}

	cfg := Config{Name: "example"}
	raw, err := jsonutil.MarshalToRawMessage(cfg)
	if err != nil {
		return err
	}

Use UnmarshalFromRawMessage to decode json.RawMessage into Go values:

	var result Config
	err := jsonutil.UnmarshalFromRawMessage(raw, &result)
	if err != nil {
		return err
	}

# Nil and Empty Handling

Both functions handle edge cases appropriately:

  - MarshalToRawMessage returns nil for nil interface values
  - UnmarshalFromRawMessage safely handles nil or empty data
  - Pointer targets are validated before unmarshaling
*/
package json
