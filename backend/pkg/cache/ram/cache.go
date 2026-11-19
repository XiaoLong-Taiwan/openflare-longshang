// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package ram provides a thin wrapper around Otter v2 for process-local caching.
package ram

import (
	"iter"
	"time"

	"github.com/maypok86/otter/v2"
)

const defaultMaximumSize = 256

// Options configures a RAM cache instance.
type Options struct {
	// MaximumSize bounds the number of entries. Zero uses a small default.
	MaximumSize int
	// TTL expires entries after their most recent write. Zero disables expiry.
	TTL time.Duration
}

// Cache is a concurrency-safe in-memory cache backed by Otter.
type Cache[K comparable, V any] struct {
	inner *otter.Cache[K, V]
}

// New creates a RAM cache from the provided options.
func New[K comparable, V any](opts Options) (*Cache[K, V], error) {
	maximumSize := opts.MaximumSize
	if maximumSize == 0 {
		maximumSize = defaultMaximumSize
	}

	options := &otter.Options[K, V]{MaximumSize: maximumSize}
	if opts.TTL > 0 {
		options.ExpiryCalculator = otter.ExpiryWriting[K, V](opts.TTL)
	}
	inner, err := otter.New(options)
	if err != nil {
		return nil, err
	}

	return &Cache[K, V]{inner: inner}, nil
}

// MustNew creates a RAM cache and panics when configuration is invalid.
func MustNew[K comparable, V any](opts Options) *Cache[K, V] {
	cache, err := New[K, V](opts)
	if err != nil {
		panic(err)
	}
	return cache
}

// GetIfPresent returns the cached value when present.
func (c *Cache[K, V]) GetIfPresent(key K) (V, bool) {
	return c.inner.GetIfPresent(key)
}

// Set stores a value in the cache.
func (c *Cache[K, V]) Set(key K, value V) {
	c.inner.Set(key, value)
}

// Invalidate removes one entry from the cache.
func (c *Cache[K, V]) Invalidate(key K) {
	c.inner.Invalidate(key)
}

// InvalidateAll removes every entry from the cache.
func (c *Cache[K, V]) InvalidateAll() {
	c.inner.InvalidateAll()
}

// Keys iterates over the keys of live entries in no particular order.
func (c *Cache[K, V]) Keys() iter.Seq[K] {
	return c.inner.Keys()
}

// CleanUp removes expired entries and runs pending eviction maintenance.
func (c *Cache[K, V]) CleanUp() {
	c.inner.CleanUp()
}

// EstimatedSize returns the approximate number of cached entries.
func (c *Cache[K, V]) EstimatedSize() int {
	return c.inner.EstimatedSize()
}
