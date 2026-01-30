// Copyright (c) Microsoft. All rights reserved.

package observability

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTracer_ReturnsTracer(t *testing.T) {
	// Act
	tracer := Tracer()

	// Assert
	assert.NotNil(t, tracer, "Tracer should not be nil")
}

func TestMeter_ReturnsMeter(t *testing.T) {
	// Act
	meter := Meter()

	// Assert
	assert.NotNil(t, meter, "Meter should not be nil")
}
