package cache_memory

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"Wavelet/core/contracts"
)

func TestTakeOnceAndExpired(t *testing.T) {
	svc, err := newMemoryCacheService(100, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := svc.Set(ctx, "state", "value", time.Minute); err != nil {
		t.Fatal(err)
	}
	var winners atomic.Int32
	var wg sync.WaitGroup
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var value string
			if err := svc.Take(ctx, "state", &value); err == nil {
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
	svc.ramCache.Set("expired", ramEntry{data: []byte(`"value"`), expireAt: time.Now().Add(-time.Second)})
	var value string
	if err := svc.Take(ctx, "expired", &value); !errors.Is(err, contracts.ErrCacheMiss) {
		t.Errorf("Take(expired) = %v, want cache miss", err)
	}
}
