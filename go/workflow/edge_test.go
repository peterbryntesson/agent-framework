// Copyright (c) Microsoft. All rights reserved.

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEdge(t *testing.T) {
	t.Run("NewEdge creates direct edge", func(t *testing.T) {
		// Act
		edge := NewEdge("a", "b")

		// Assert
		assert.Equal(t, "a", edge.From)
		assert.Equal(t, "b", edge.To)
		assert.Nil(t, edge.Condition)
	})

	t.Run("NewConditionalEdge creates edge with condition", func(t *testing.T) {
		// Arrange
		condition := func(state map[string]interface{}) bool {
			return state["proceed"] == true
		}

		// Act
		edge := NewConditionalEdge("a", "b", condition)

		// Assert
		assert.Equal(t, "a", edge.From)
		assert.Equal(t, "b", edge.To)
		assert.NotNil(t, edge.Condition)

		// Verify condition works
		assert.True(t, edge.Condition(map[string]interface{}{"proceed": true}))
		assert.False(t, edge.Condition(map[string]interface{}{"proceed": false}))
	})
}

func TestFanOutEdge(t *testing.T) {
	t.Run("creates fan-out group", func(t *testing.T) {
		// Act
		group := FanOutEdge("source", "target1", "target2", "target3")

		// Assert
		assert.Equal(t, EdgeGroupTypeFanOut, group.Type)
		assert.Equal(t, "source", group.From)
		assert.Equal(t, []string{"target1", "target2", "target3"}, group.Targets)
	})

	t.Run("works with single target", func(t *testing.T) {
		// Act
		group := FanOutEdge("source", "target1")

		// Assert
		assert.Equal(t, EdgeGroupTypeFanOut, group.Type)
		assert.Len(t, group.Targets, 1)
	})
}

func TestFanInEdge(t *testing.T) {
	t.Run("creates fan-in group", func(t *testing.T) {
		// Act
		group := FanInEdge([]string{"source1", "source2", "source3"}, "target")

		// Assert
		assert.Equal(t, EdgeGroupTypeFanIn, group.Type)
		assert.Equal(t, "target", group.To)
		assert.Equal(t, []string{"source1", "source2", "source3"}, group.Sources)
	})
}

func TestSwitchEdge(t *testing.T) {
	t.Run("creates switch edge with selector", func(t *testing.T) {
		// Arrange
		selector := func(state map[string]interface{}) interface{} {
			return state["choice"]
		}

		// Act
		sw := NewSwitchEdge("router", selector)

		// Assert
		assert.Equal(t, "router", sw.From)
		assert.NotNil(t, sw.Selector)
		assert.Empty(t, sw.Cases)
		assert.Empty(t, sw.Default)
	})

	t.Run("Case adds cases fluently", func(t *testing.T) {
		// Arrange
		selector := func(state map[string]interface{}) interface{} {
			return state["choice"]
		}

		// Act
		sw := NewSwitchEdge("router", selector).
			Case("a", "target-a").
			Case("b", "target-b").
			Case("c", "target-c")

		// Assert
		assert.Len(t, sw.Cases, 3)
		assert.Equal(t, "a", sw.Cases[0].CaseValue)
		assert.Equal(t, "target-a", sw.Cases[0].Target)
		assert.Equal(t, "b", sw.Cases[1].CaseValue)
		assert.Equal(t, "target-b", sw.Cases[1].Target)
	})

	t.Run("DefaultCase sets default target", func(t *testing.T) {
		// Act
		sw := NewSwitchEdge("router", nil).
			Case("known", "known-target").
			DefaultCase("fallback")

		// Assert
		assert.Equal(t, "fallback", sw.Default)
	})

	t.Run("Evaluate returns matching case", func(t *testing.T) {
		// Arrange
		selector := func(state map[string]interface{}) interface{} {
			return state["choice"]
		}
		sw := NewSwitchEdge("router", selector).
			Case("a", "target-a").
			Case("b", "target-b").
			DefaultCase("default-target")

		// Act & Assert
		assert.Equal(t, "target-a", sw.Evaluate(map[string]interface{}{"choice": "a"}))
		assert.Equal(t, "target-b", sw.Evaluate(map[string]interface{}{"choice": "b"}))
		assert.Equal(t, "default-target", sw.Evaluate(map[string]interface{}{"choice": "unknown"}))
	})

	t.Run("Evaluate returns default when no selector", func(t *testing.T) {
		// Arrange
		sw := NewSwitchEdge("router", nil).
			Case("a", "target-a").
			DefaultCase("default-target")

		// Act
		result := sw.Evaluate(map[string]interface{}{"choice": "a"})

		// Assert
		assert.Equal(t, "default-target", result)
	})

	t.Run("Evaluate handles numeric cases", func(t *testing.T) {
		// Arrange
		selector := func(state map[string]interface{}) interface{} {
			return state["code"]
		}
		sw := NewSwitchEdge("router", selector).
			Case(1, "one").
			Case(2, "two").
			Case(3, "three")

		// Act & Assert
		assert.Equal(t, "one", sw.Evaluate(map[string]interface{}{"code": 1}))
		assert.Equal(t, "two", sw.Evaluate(map[string]interface{}{"code": 2}))
		assert.Equal(t, "", sw.Evaluate(map[string]interface{}{"code": 99}))
	})
}

func TestEdgeGroupTypes(t *testing.T) {
	t.Run("EdgeGroupType constants are distinct", func(t *testing.T) {
		// Assert
		assert.NotEqual(t, EdgeGroupTypeFanOut, EdgeGroupTypeFanIn)
		assert.NotEqual(t, EdgeGroupTypeFanOut, EdgeGroupTypeSwitch)
		assert.NotEqual(t, EdgeGroupTypeFanIn, EdgeGroupTypeSwitch)
	})
}
