// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package node

import (
	"errors"
	"testing"

	"gorm.io/gorm"
)

func TestSafeBatchErrorDoesNotExposeUnderlyingError(t *testing.T) {
	if got := safeBatchError(errors.New("database password leaked")); got != "節點操作失敗" {
		t.Fatalf("safeBatchError() = %q, want generic message", got)
	}
	if got := safeBatchError(gorm.ErrRecordNotFound); got != errNodeNotFound {
		t.Fatalf("safeBatchError(record not found) = %q, want %q", got, errNodeNotFound)
	}
}
