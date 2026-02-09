// Copyright (c) Microsoft. All rights reserved.

package openai

import (
	"time"

	"github.com/google/uuid"
)

func newID() string {
	return uuid.NewString()
}

func nowUnix() int64 {
	return time.Now().Unix()
}
