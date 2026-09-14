// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package task

import (
	"context"
	"testing"
)

// This file exists so `go test` on the task adapter stays valid.
func TestSetServiceNil(t *testing.T) {
	SetService(nil)
	if _, err := DispatchTask(context.Background(), "x", nil, "test"); err == nil {
		t.Fatal("expected error when task service is nil")
	}
}
