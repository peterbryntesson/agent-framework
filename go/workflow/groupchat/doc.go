// Copyright (c) Microsoft. All rights reserved.

// Package groupchat provides multi-agent conversation orchestration.
//
// The groupchat package enables multiple AI agents to participate in coordinated
// conversations, with pluggable strategies for selecting which agent speaks next.
//
// Key components:
//   - Selector: Interface for speaker selection strategies
//   - Transcript: Records conversation history with speaker attribution
//   - Manager: Orchestrates the group chat execution loop
//
// Built-in selectors:
//   - RoundRobinSelector: Cycles through agents in order
//   - RandomSelector: Picks a random agent
//   - LLMSelector: Uses an LLM to decide the next speaker
//
// Basic usage:
//
//	agents := []agent.Agent{agentA, agentB, agentC}
//	manager := groupchat.NewManager(
//	    agents,
//	    groupchat.WithSelector(groupchat.NewRoundRobinSelector(agents...)),
//	    groupchat.WithMaxTurns(10),
//	)
//
//	transcript, err := manager.Run(ctx, "Start the discussion")
//
// Streaming usage:
//
//	events, err := manager.RunStream(ctx, "Start the discussion")
//	for event := range events {
//	    switch event.Kind {
//	    case groupchat.EventKindTurn:
//	        fmt.Printf("%s: %s\n", event.Speaker, event.Message)
//	    case groupchat.EventKindCompleted:
//	        fmt.Println("Chat completed")
//	    }
//	}
package groupchat
