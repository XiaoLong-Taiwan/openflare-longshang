// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package cache_test

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"Wavelet/plugins/infra/cache"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCachePluginOperations(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	defer func() { _ = rdb.Close() }()

	p := cache.New(
		cache.WithRedis(rdb),
		cache.WithKeyPrefix("app:"),
		cache.WithRAMCapacity(500),
	)
	ctx := core.NewContext(context.Background())
	ctx.Config().SetSource(core.NewMapSource(map[string]any{
		"redis.enabled": true,
	}))
	require.NoError(t, ctx.Config().Resolve())
	require.NoError(t, p.Apply(ctx))

	svc, err := core.Inject[contracts.CacheService](ctx)
	require.NoError(t, err)

	type Data struct {
		Value string `json:"value"`
	}

	testCtx := context.Background()

	// 1. ErrCacheMiss
	var out Data
	err = svc.Get(testCtx, "missing", &out)
	assert.ErrorIs(t, err, contracts.ErrCacheMiss)

	// 2. Set & Get
	in := Data{Value: "hello"}
	require.NoError(t, svc.Set(testCtx, "key1", in, 5*time.Minute))

	require.NoError(t, svc.Get(testCtx, "key1", &out))
	assert.Equal(t, "hello", out.Value)

	// 3. GetOrSet
	var target Data
	err = svc.GetOrSet(testCtx, "key1", &target, time.Minute, func() (any, error) {
		return Data{Value: "from_loader"}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "hello", target.Value)

	err = svc.GetOrSet(testCtx, "key2", &target, time.Minute, func() (any, error) {
		return Data{Value: "from_loader"}, nil
	})
	require.NoError(t, err)
	assert.Equal(t, "from_loader", target.Value)

	// 4. Invalidate / Delete
	require.NoError(t, svc.Invalidate(testCtx, "key1"))
	err = svc.Get(testCtx, "key1", &out)
	assert.ErrorIs(t, err, contracts.ErrCacheMiss)

	require.NoError(t, ctx.Dispose())
}

func TestTakeAcrossNodes(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	services := make([]contracts.CacheService, 2)
	for i := range services {
		ctx := core.NewContext(context.Background())
		ctx.Config().SetSource(core.NewMapSource(map[string]any{"redis.enabled": true}))
		require.NoError(t, ctx.Config().Resolve())
		require.NoError(t, cache.New(cache.WithRedis(rdb), cache.WithKeyPrefix("take:")).Apply(ctx))
		t.Cleanup(func() { _ = ctx.Dispose() })
		services[i], _ = core.Inject[contracts.CacheService](ctx)
	}
	ctx := context.Background()
	subscriber := rdb.Subscribe(ctx, "wavelet:cache:invalidation")
	t.Cleanup(func() { _ = subscriber.Close() })
	if _, err := subscriber.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	require.NoError(t, services[0].Set(ctx, "broadcast", "value", time.Minute))
	var consumed string
	require.NoError(t, services[0].Take(ctx, "broadcast", &consumed))
	readCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	message, err := subscriber.ReceiveMessage(readCtx)
	if err != nil || message.Payload != "broadcast" {
		t.Fatalf("Take(broadcast) message = %v, error = %v, want invalidation", message, err)
	}
	require.NoError(t, services[0].Set(ctx, "state", "verifier", time.Minute))
	var out string
	require.NoError(t, services[1].Get(ctx, "state", &out))
	var winners atomic.Int32
	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var value string
			if err := services[i%2].Take(ctx, "state", &value); err == nil {
				winners.Add(1)
			} else if !errors.Is(err, contracts.ErrCacheMiss) {
				t.Errorf("Take(state) error = %v, want cache miss", err)
			}
		}()
	}
	wg.Wait()
	if winners.Load() != 1 {
		t.Errorf("Take(state) winners = %d, want 1", winners.Load())
	}
	require.NoError(t, services[0].Set(ctx, "expired", "value", time.Second))
	mr.FastForward(2 * time.Second)
	if err := services[0].Take(ctx, "expired", &out); !errors.Is(err, contracts.ErrCacheMiss) {
		t.Errorf("Take(expired with warm RAM) = %v, want cache miss", err)
	}
}
