// Copyright (c) Microsoft. All rights reserved.

package devui

import (
	"container/list"
	"sync"
	"time"

	"go.opentelemetry.io/otel/sdk/trace"
)

// TraceCollector collects and stores OpenTelemetry spans for visualization.
// It implements trace.SpanExporter for integration with the OTEL SDK.
type TraceCollector struct {
	maxTraces int
	traces    map[string]*collectedTrace
	order     *list.List // LRU order for eviction
	mu        sync.RWMutex
}

type collectedTrace struct {
	info     TraceInfo
	spans    []SpanInfo
	element  *list.Element // Position in LRU list
	complete bool
}

// NewTraceCollector creates a new trace collector with the specified capacity.
func NewTraceCollector(maxTraces int) *TraceCollector {
	if maxTraces <= 0 {
		maxTraces = 1000
	}
	return &TraceCollector{
		maxTraces: maxTraces,
		traces:    make(map[string]*collectedTrace),
		order:     list.New(),
	}
}

// ExportSpans implements trace.SpanExporter.
// It receives completed spans and organizes them by trace ID.
func (c *TraceCollector) ExportSpans(_ interface{}, spans []trace.ReadOnlySpan) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, span := range spans {
		traceID := span.SpanContext().TraceID().String()

		// Get or create trace
		tr, exists := c.traces[traceID]
		if !exists {
			// Evict oldest if at capacity
			if len(c.traces) >= c.maxTraces {
				c.evictOldest()
			}

			tr = &collectedTrace{
				info: TraceInfo{
					TraceID:   traceID,
					StartTime: span.StartTime(),
					Status:    "in_progress",
				},
				spans: make([]SpanInfo, 0),
			}
			tr.element = c.order.PushFront(traceID)
			c.traces[traceID] = tr
		} else {
			// Move to front (most recently used)
			c.order.MoveToFront(tr.element)
		}

		// Extract agent name from attributes
		for _, attr := range span.Attributes() {
			if string(attr.Key) == "agent.name" || string(attr.Key) == "gen_ai.agent.name" {
				tr.info.AgentName = attr.Value.AsString()
				break
			}
		}

		// Convert attributes to map
		attrs := make(map[string]string)
		for _, attr := range span.Attributes() {
			attrs[string(attr.Key)] = attr.Value.Emit()
		}

		// Convert events
		events := make([]SpanEvent, 0, len(span.Events()))
		for _, ev := range span.Events() {
			eventAttrs := make(map[string]string)
			for _, attr := range ev.Attributes {
				eventAttrs[string(attr.Key)] = attr.Value.Emit()
			}
			events = append(events, SpanEvent{
				Name:       ev.Name,
				Timestamp:  ev.Time,
				Attributes: eventAttrs,
			})
		}

		// Add span info
		spanInfo := SpanInfo{
			SpanID:     span.SpanContext().SpanID().String(),
			Name:       span.Name(),
			StartTime:  span.StartTime(),
			EndTime:    span.EndTime(),
			Duration:   span.EndTime().Sub(span.StartTime()),
			Status:     span.Status().Code.String(),
			Attributes: attrs,
			Events:     events,
		}

		if span.Parent().IsValid() {
			spanInfo.ParentSpanID = span.Parent().SpanID().String()
		}

		tr.spans = append(tr.spans, spanInfo)
		tr.info.SpanCount = len(tr.spans)

		// Update trace timing
		if span.StartTime().Before(tr.info.StartTime) {
			tr.info.StartTime = span.StartTime()
		}
		if span.EndTime().After(tr.info.EndTime) {
			tr.info.EndTime = span.EndTime()
			tr.info.Duration = tr.info.EndTime.Sub(tr.info.StartTime)
		}

		// Update status based on span status
		if span.Status().Code.String() == "Error" {
			tr.info.Status = "error"
		} else if !tr.complete && span.Parent().SpanID().IsValid() == false {
			// Root span completed
			tr.info.Status = "completed"
			tr.complete = true
		}
	}

	return nil
}

// Shutdown implements trace.SpanExporter.
func (c *TraceCollector) Shutdown(_ interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.traces = make(map[string]*collectedTrace)
	c.order = list.New()
	return nil
}

// List returns info for all collected traces, ordered by start time (newest first).
func (c *TraceCollector) List() []TraceInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()

	infos := make([]TraceInfo, 0, len(c.traces))
	for e := c.order.Front(); e != nil; e = e.Next() {
		traceID := e.Value.(string)
		if tr, ok := c.traces[traceID]; ok {
			infos = append(infos, tr.info)
		}
	}
	return infos
}

// Get retrieves full details for a specific trace.
// Returns zero value and false if not found.
func (c *TraceCollector) Get(traceID string) (TraceDetail, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if tr, ok := c.traces[traceID]; ok {
		return TraceDetail{
			TraceInfo: tr.info,
			Spans:     tr.spans,
		}, true
	}
	return TraceDetail{}, false
}

// Clear removes all collected traces.
func (c *TraceCollector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.traces = make(map[string]*collectedTrace)
	c.order = list.New()
}

// Count returns the number of stored traces.
func (c *TraceCollector) Count() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.traces)
}

// evictOldest removes the least recently used trace.
// Must be called with lock held.
func (c *TraceCollector) evictOldest() {
	if c.order.Len() == 0 {
		return
	}
	oldest := c.order.Back()
	if oldest != nil {
		traceID := oldest.Value.(string)
		delete(c.traces, traceID)
		c.order.Remove(oldest)
	}
}

// MarkComplete marks a trace as completed.
// This is useful for explicitly ending traces that may not have a root span.
func (c *TraceCollector) MarkComplete(traceID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if tr, ok := c.traces[traceID]; ok {
		tr.complete = true
		if tr.info.Status == "in_progress" {
			tr.info.Status = "completed"
		}
		if tr.info.EndTime.IsZero() {
			tr.info.EndTime = time.Now()
			tr.info.Duration = tr.info.EndTime.Sub(tr.info.StartTime)
		}
	}
}
