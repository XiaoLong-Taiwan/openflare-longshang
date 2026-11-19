// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package observability

import (
	"testing"
	"time"

	"Wavelet/pkg/cache/ram"
)

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

func TestNodeObservabilityResponseCacheExpiresEntries(t *testing.T) {
	cache := &nodeObservabilityResponseCache{views: ram.MustNew[string, *NodeView](ram.Options{
		MaximumSize: 2,
		TTL:         10 * time.Millisecond,
	})}
	cache.views.Set("node-a:edge_node:24:120", &NodeView{NodeID: "node-a"})
	time.Sleep(20 * time.Millisecond)
	if _, ok := cache.views.GetIfPresent("node-a:edge_node:24:120"); ok {
		t.Fatal("expired observability entry is still present")
	}
}

func TestNodeObservabilityResponseCacheIsBounded(t *testing.T) {
	cache := &nodeObservabilityResponseCache{views: ram.MustNew[string, *NodeView](ram.Options{MaximumSize: 2})}
	cache.views.Set("node-a:edge_node:24:120", &NodeView{NodeID: "node-a"})
	cache.views.Set("node-b:edge_node:24:120", &NodeView{NodeID: "node-b"})
	cache.views.Set("node-c:edge_node:24:120", &NodeView{NodeID: "node-c"})
	cache.views.CleanUp()
	if got := cache.views.EstimatedSize(); got > 2 {
		t.Fatalf("cache size = %d, want at most 2", got)
	}
}

func TestNodeObservabilityResponseCacheInvalidatesOnlyNodeEntries(t *testing.T) {
	cache := &nodeObservabilityResponseCache{views: ram.MustNew[string, *NodeView](ram.Options{MaximumSize: 3})}
	cache.views.Set("node-a:edge_node:24:120", &NodeView{NodeID: "node-a"})
	cache.views.Set("node-a:tunnel_relay:24:120", &NodeView{NodeID: "node-a"})
	cache.views.Set("node-b:edge_node:24:120", &NodeView{NodeID: "node-b"})

	cache.invalidateNode("node-a")
	if _, ok := cache.views.GetIfPresent("node-a:edge_node:24:120"); ok {
		t.Fatal("node-a edge entry was not invalidated")
	}
	if _, ok := cache.views.GetIfPresent("node-a:tunnel_relay:24:120"); ok {
		t.Fatal("node-a relay entry was not invalidated")
	}
	if _, ok := cache.views.GetIfPresent("node-b:edge_node:24:120"); !ok {
		t.Fatal("node-b entry was invalidated unexpectedly")
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

func TestNodeObservabilityCacheDoesNotRepopulateAfterInvalidation(t *testing.T) {
	cache := newNodeObservabilityCache()
	key := nodeObservabilityCacheKey("node-a", 24, 120, "edge_node")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})
	stale := &NodeView{NodeID: "node-a"}
	go func() {
		defer close(done)
		_, err := cache.getOrLoad(t.Context(), key, func() (*NodeView, error) {
			close(started)
			<-release
			return stale, nil
		})
		if err != nil {
			t.Errorf("getOrLoad(stale) error = %v, want nil", err)
		}
	}()
	<-started
	cache.invalidateNode("node-a")
	fresh := &NodeView{NodeID: "node-a", HealthEvents: nil}
	got, err := cache.getOrLoad(t.Context(), key, func() (*NodeView, error) { return fresh, nil })
	close(release)
	<-done
	if err != nil || got != fresh {
		t.Fatalf("getOrLoad(fresh) = %p, %v, want %p, nil", got, err, fresh)
	}
	if cached, ok := cache.views.GetIfPresent(key); !ok || cached != fresh {
		t.Errorf("cache after stale loader = %p, %t, want %p, true", cached, ok, fresh)
	}
	cache.invalidateNode("node-a")
	_, err = cache.getOrLoad(t.Context(), key, func() (*NodeView, error) {
		cache.invalidateNode("node-a")
		return stale, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.views.GetIfPresent(key); ok {
		t.Error("cache after invalidation during loader is populated, want empty")
	}
}
