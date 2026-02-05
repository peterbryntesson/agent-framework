// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTraceCollector_New(t *testing.T) {
	collector := NewTraceCollector(100)

	assert.NotNil(t, collector)
	assert.Equal(t, 0, collector.Count())
}

func TestTraceCollector_DefaultMaxTraces(t *testing.T) {
	collector := NewTraceCollector(0)

	assert.NotNil(t, collector)
	assert.Equal(t, 1000, collector.maxTraces)
}

func TestTraceCollector_List_Empty(t *testing.T) {
	collector := NewTraceCollector(10)

	traces := collector.List()

	assert.Empty(t, traces)
}

func TestTraceCollector_Get_NotFound(t *testing.T) {
	collector := NewTraceCollector(10)

	detail, ok := collector.Get("nonexistent-trace-id")

	assert.False(t, ok)
	assert.Empty(t, detail.TraceID)
}

func TestTraceCollector_Clear(t *testing.T) {
	collector := NewTraceCollector(10)

	// Add some mock trace data manually
	collector.mu.Lock()
	collector.traces["test-trace"] = &collectedTrace{
		info: TraceInfo{TraceID: "test-trace"},
	}
	collector.order.PushFront("test-trace")
	collector.mu.Unlock()

	assert.Equal(t, 1, collector.Count())

	collector.Clear()

	assert.Equal(t, 0, collector.Count())
}

func TestTraceCollector_MarkComplete(t *testing.T) {
	collector := NewTraceCollector(10)

	// Add a trace manually
	collector.mu.Lock()
	collector.traces["test-trace"] = &collectedTrace{
		info: TraceInfo{
			TraceID:   "test-trace",
			Status:    "in_progress",
			StartTime: time.Now().Add(-time.Minute),
		},
		complete: false,
	}
	collector.order.PushFront("test-trace")
	collector.mu.Unlock()

	collector.MarkComplete("test-trace")

	collector.mu.RLock()
	tr := collector.traces["test-trace"]
	collector.mu.RUnlock()

	assert.True(t, tr.complete)
	assert.Equal(t, "completed", tr.info.Status)
	assert.False(t, tr.info.EndTime.IsZero())
}

func TestTraceCollector_MarkComplete_NonExistent(t *testing.T) {
	collector := NewTraceCollector(10)

	// Should not panic
	collector.MarkComplete("nonexistent")
}

func TestTraceCollector_Eviction(t *testing.T) {
	collector := NewTraceCollector(2) // Very small capacity

	// Add traces manually to test eviction
	collector.mu.Lock()
	for i := 0; i < 3; i++ {
		traceID := string(rune('a' + i))
		collector.traces[traceID] = &collectedTrace{
			info:  TraceInfo{TraceID: traceID},
			spans: make([]SpanInfo, 0),
		}
		elem := collector.order.PushFront(traceID)
		collector.traces[traceID].element = elem

		// Evict if needed
		if len(collector.traces) > collector.maxTraces {
			collector.evictOldest()
		}
	}
	collector.mu.Unlock()

	// Should only have 2 traces
	assert.Equal(t, 2, collector.Count())

	// The oldest trace (a) should be evicted
	_, ok := collector.Get("a")
	assert.False(t, ok)
}

func TestSpanInfo_Duration(t *testing.T) {
	start := time.Now()
	end := start.Add(500 * time.Millisecond)

	span := SpanInfo{
		StartTime: start,
		EndTime:   end,
		Duration:  end.Sub(start),
	}

	assert.Equal(t, 500*time.Millisecond, span.Duration)
}

func TestTraceInfo_Fields(t *testing.T) {
	info := TraceInfo{
		TraceID:   "abc123",
		AgentName: "test-agent",
		StartTime: time.Now(),
		Status:    "completed",
		SpanCount: 5,
	}

	assert.Equal(t, "abc123", info.TraceID)
	assert.Equal(t, "test-agent", info.AgentName)
	assert.Equal(t, "completed", info.Status)
	assert.Equal(t, 5, info.SpanCount)
}
