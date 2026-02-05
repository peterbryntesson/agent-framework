// Copyright (c) Microsoft. All rights reserved.

// Package agui provides AG-UI protocol types and server implementation for
// streaming agent responses to UI clients via Server-Sent Events (SSE).
//
// AG-UI (Agent User Interface) is a protocol for real-time communication between
// AI agents and user interface clients. It defines a standardized set of events
// for streaming agent responses, including text messages, tool calls, and state updates.
//
// # Event Types
//
// The protocol defines the following event types:
//
//   - RUN_STARTED: Indicates the start of an agent run
//   - RUN_FINISHED: Indicates successful completion of a run
//   - RUN_ERROR: Indicates an error occurred during the run
//   - TEXT_MESSAGE_START: Marks the beginning of a text message
//   - TEXT_MESSAGE_CONTENT: Contains incremental text content (delta)
//   - TEXT_MESSAGE_END: Marks the end of a text message
//   - TOOL_CALL_START: Marks the beginning of a tool call
//   - TOOL_CALL_ARGS: Contains incremental tool call arguments (delta)
//   - TOOL_CALL_END: Marks the end of a tool call
//   - TOOL_CALL_RESULT: Contains the result of a tool call
//   - STATE_SNAPSHOT: Contains a complete state snapshot
//   - STATE_DELTA: Contains an incremental state update
//
// # Server Usage
//
// The Server type exposes an agent via AG-UI protocol endpoints:
//
//	agent := myAgent // implements agent.Agent
//	server := agui.NewServer(agent)
//	http.Handle("/agui/", server.Handler())
//
// # Event Conversion
//
// The EventConverter type converts agent.ResponseUpdate values to AG-UI events:
//
//	converter := agui.NewEventConverter("thread-123", "run-456")
//	events := converter.ConvertAll(responseUpdates)
//
// Events are serialized to JSON and sent via SSE with the following format:
//
//	event: <event-type>
//	data: <json-payload>
//
// # Protocol Specification
//
// For the full AG-UI protocol specification, see: https://docs.ag-ui.com/
package agui
