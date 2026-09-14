// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package ofutil

import "strings"

// UniqueAndCleanStringSlice trims spaces, drops empties, and de-duplicates
// while preserving order. An empty result is nil.
func UniqueAndCleanStringSlice(slice []string) []string {
	if slice == nil {
		return nil
	}
	seen := make(map[string]struct{})
	result := make([]string, 0)
	for _, item := range slice {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
