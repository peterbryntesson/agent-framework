// Copyright (c) Microsoft. All rights reserved.

package workflow

// EdgeCondition is a function that determines if an edge should be traversed.
// The state parameter contains the workflow's current state.
// Return true to allow message routing through this edge.
type EdgeCondition func(state map[string]interface{}) bool

// Edge defines a connection between two executors in a workflow.
type Edge struct {
	// From is the source executor ID
	From string

	// To is the target executor ID
	To string

	// Condition optionally guards message routing.
	// If nil, messages always flow through this edge.
	Condition EdgeCondition
}

// NewEdge creates a direct edge between two executors.
func NewEdge(from, to string) Edge {
	return Edge{From: from, To: to}
}

// NewConditionalEdge creates an edge with a routing condition.
func NewConditionalEdge(from, to string, condition EdgeCondition) Edge {
	return Edge{From: from, To: to, Condition: condition}
}

// EdgeGroup represents a collection of related edges for fan-out/fan-in patterns.
type EdgeGroup struct {
	// Type indicates the edge group pattern
	Type EdgeGroupType

	// From is the source executor ID (for fan-out)
	From string

	// To is the target executor ID (for fan-in)
	To string

	// Targets are the destination executor IDs (for fan-out)
	Targets []string

	// Sources are the source executor IDs (for fan-in)
	Sources []string
}

// EdgeGroupType defines the type of edge group pattern.
type EdgeGroupType int

const (
	// EdgeGroupTypeFanOut routes from one executor to multiple targets
	EdgeGroupTypeFanOut EdgeGroupType = iota

	// EdgeGroupTypeFanIn collects from multiple sources to one target
	EdgeGroupTypeFanIn

	// EdgeGroupTypeSwitch routes based on case matching
	EdgeGroupTypeSwitch
)

// FanOutEdge creates an edge group that routes from one executor to multiple targets.
func FanOutEdge(from string, targets ...string) EdgeGroup {
	return EdgeGroup{
		Type:    EdgeGroupTypeFanOut,
		From:    from,
		Targets: targets,
	}
}

// FanInEdge creates an edge group that collects from multiple sources to one target.
func FanInEdge(sources []string, to string) EdgeGroup {
	return EdgeGroup{
		Type:    EdgeGroupTypeFanIn,
		To:      to,
		Sources: sources,
	}
}

// SwitchCase represents a case in a switch edge.
type SwitchCase struct {
	// CaseValue is the value to match
	CaseValue interface{}

	// Target is the executor ID to route to when matched
	Target string
}

// SwitchEdge represents a switch/case routing pattern.
type SwitchEdge struct {
	// From is the source executor ID
	From string

	// Selector extracts the value to match from state
	Selector func(state map[string]interface{}) interface{}

	// Cases define the routing rules
	Cases []SwitchCase

	// Default is the fallback target if no case matches
	Default string
}

// NewSwitchEdge creates a switch edge with the given selector and cases.
func NewSwitchEdge(from string, selector func(state map[string]interface{}) interface{}) *SwitchEdge {
	return &SwitchEdge{
		From:     from,
		Selector: selector,
		Cases:    make([]SwitchCase, 0),
	}
}

// Case adds a case to the switch edge.
func (se *SwitchEdge) Case(value interface{}, target string) *SwitchEdge {
	se.Cases = append(se.Cases, SwitchCase{CaseValue: value, Target: target})
	return se
}

// DefaultCase sets the default target for the switch edge.
func (se *SwitchEdge) DefaultCase(target string) *SwitchEdge {
	se.Default = target
	return se
}

// Evaluate determines the target executor based on the current state.
func (se *SwitchEdge) Evaluate(state map[string]interface{}) string {
	if se.Selector == nil {
		return se.Default
	}
	value := se.Selector(state)
	for _, c := range se.Cases {
		if c.CaseValue == value {
			return c.Target
		}
	}
	return se.Default
}
