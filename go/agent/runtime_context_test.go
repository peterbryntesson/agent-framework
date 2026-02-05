// Copyright (c) Microsoft. All rights reserved.

package agent

import (
	"context"
	"testing"
)

func TestRuntimeContext_NewAndGet(t *testing.T) {
	rtc := NewRuntimeContext()

	_, ok := rtc.Get("missing")
	if ok {
		t.Error("Get should return false for missing key")
	}

	rtc2 := rtc.With("user_id", "u123")
	val, ok := rtc2.Get("user_id")
	if !ok || val != "u123" {
		t.Errorf("Get() = %v, %v; want u123, true", val, ok)
	}

	// Original unchanged
	_, ok = rtc.Get("user_id")
	if ok {
		t.Error("With should not modify original context")
	}
}

func TestRuntimeContext_Values(t *testing.T) {
	rtc := NewRuntimeContext().
		With("a", "1").
		With("b", "2")

	vals := rtc.Values()
	if len(vals) != 2 {
		t.Errorf("Values() len = %d; want 2", len(vals))
	}
	if vals["a"] != "1" || vals["b"] != "2" {
		t.Error("Values() content mismatch")
	}
}

func TestRuntimeContext_Len(t *testing.T) {
	rtc := NewRuntimeContext()
	if rtc.Len() != 0 {
		t.Errorf("Len() = %d; want 0", rtc.Len())
	}

	rtc = rtc.With("key", "value")
	if rtc.Len() != 1 {
		t.Errorf("Len() = %d; want 1", rtc.Len())
	}
}

func TestRuntimeContext_Immutability(t *testing.T) {
	rtc1 := NewRuntimeContext().With("key1", "value1")
	rtc2 := rtc1.With("key2", "value2")

	// rtc1 should still have only 1 key
	if rtc1.Len() != 1 {
		t.Errorf("rtc1.Len() = %d; want 1 (immutability violated)", rtc1.Len())
	}

	// rtc2 should have 2 keys
	if rtc2.Len() != 2 {
		t.Errorf("rtc2.Len() = %d; want 2", rtc2.Len())
	}

	// Verify values don't leak
	_, ok := rtc1.Get("key2")
	if ok {
		t.Error("key2 should not be in rtc1")
	}
}

func TestRuntimeCtxFromContext_NoContext(t *testing.T) {
	ctx := context.Background()

	rtc := RuntimeCtxFromContext(ctx)
	if rtc == nil {
		t.Fatal("Should return non-nil RuntimeContext")
	}
	if rtc.Len() != 0 {
		t.Error("Should return empty RuntimeContext when not set")
	}
}

func TestRuntimeCtxFromContext_WithContext(t *testing.T) {
	ctx := context.Background()

	rtc := NewRuntimeContext().With("key", "value")
	ctx = WithRuntimeCtx(ctx, rtc)

	extracted := RuntimeCtxFromContext(ctx)
	val, ok := extracted.Get("key")
	if !ok || val != "value" {
		t.Errorf("Should extract RuntimeContext from context.Context; got %v, %v", val, ok)
	}
}

func TestRuntimeCtxFromContext_NilContext(t *testing.T) {
	ctx := context.Background()

	// Explicitly set nil
	ctx = context.WithValue(ctx, runtimeCtxKey{}, (*RuntimeContext)(nil))

	rtc := RuntimeCtxFromContext(ctx)
	if rtc == nil {
		t.Fatal("Should return non-nil RuntimeContext even when nil is stored")
	}
	if rtc.Len() != 0 {
		t.Error("Should return empty RuntimeContext when nil is stored")
	}
}

func TestRuntimeContext_ValuesIsolation(t *testing.T) {
	rtc := NewRuntimeContext().With("key", "value")

	// Get values and modify the returned map
	vals := rtc.Values()
	vals["key"] = "modified"
	vals["new_key"] = "new_value"

	// Original should be unchanged
	val, _ := rtc.Get("key")
	if val != "value" {
		t.Error("Values() should return a copy, not the original map")
	}

	_, ok := rtc.Get("new_key")
	if ok {
		t.Error("Modifications to Values() should not affect the context")
	}
}
