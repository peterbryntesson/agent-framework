// Copyright (c) Microsoft. All rights reserved.

package declarative

import "strings"

func normalizeAPIType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeToolKind(value string) string {
	kind := strings.TrimSpace(value)
	kind = strings.ReplaceAll(kind, "_", "")
	kind = strings.ReplaceAll(kind, "-", "")
	return strings.ToLower(kind)
}
