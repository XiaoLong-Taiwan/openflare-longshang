// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package observability

import "testing"

func TestNodeObservabilityCacheKeyIncludesQueryAndNodeType(t *testing.T) {
	base := nodeObservabilityCacheKey("node-a", 24, 120, "edge_node")

	for _, tc := range []struct {
		name string
		key  string
	}{
		{name: "node", key: nodeObservabilityCacheKey("node-b", 24, 120, "edge_node")},
		{name: "hours", key: nodeObservabilityCacheKey("node-a", 48, 120, "edge_node")},
		{name: "limit", key: nodeObservabilityCacheKey("node-a", 24, 240, "edge_node")},
		{name: "node type", key: nodeObservabilityCacheKey("node-a", 24, 120, "tunnel_relay")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.key == base {
				t.Fatalf("cache key %q unexpectedly matches base key", tc.key)
			}
		})
	}
}

func TestNodeObservabilityCacheKeyUsesNormalizedQuery(t *testing.T) {
	defaultKey := nodeObservabilityCacheKey(
		"node-a",
		normalizeObservabilityHours(0),
		normalizeObservabilityLimit(0),
		"edge_node",
	)
	explicitKey := nodeObservabilityCacheKey("node-a", 24, 120, "edge_node")

	if defaultKey != explicitKey {
		t.Fatalf("normalized default key = %q, explicit default key = %q", defaultKey, explicitKey)
	}
}
