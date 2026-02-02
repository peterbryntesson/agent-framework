// Copyright (c) Microsoft. All rights reserved.

/*
Package validation provides input validation functions for the agent framework.

This internal package offers helper functions for validating common inputs
such as nil checks, empty string checks, and message validation. These
validators return structured errors with context about which field failed
and why.

# Basic Usage

Use RequireNotNil to validate that a value is not nil:

	err := validation.RequireNotNil(client, "client")
	if err != nil {
		return err
	}

Use RequireNotEmpty to validate that a string is not empty:

	err := validation.RequireNotEmpty(name, "name")
	if err != nil {
		return err
	}

Use ValidateMessages to validate a messages slice before processing:

	err := validation.ValidateMessages(messages)
	if err != nil {
		return err
	}

# Error Handling

All validation functions return a ValidationError that includes:
  - Field: The name of the field or parameter that failed validation
  - Message: A description of the validation failure
  - Err: The underlying sentinel error for error type checking

Example of checking error types:

	if err := validation.RequireNotNil(client, "client"); err != nil {
		if errors.Is(err, validation.ErrNilValue) {
			// Handle nil value case
		}
	}
*/
package validation
