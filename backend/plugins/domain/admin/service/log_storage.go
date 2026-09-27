// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"Wavelet/core"
	"Wavelet/core/contracts"
	"context"
	"errors"
	"fmt"
	"time"
)

var errLogStorageUnavailable = errors.New("log storage service is unavailable")

func LogStorageOverview(ctx context.Context) (contracts.LogStorageOverview, error) {
	storage := GetLogStorageService(ctx)
	if storage == nil {
		return contracts.LogStorageOverview{}, errLogStorageUnavailable
	}
	return storage.Overview(ctx)
}

func CleanupLogStorage(ctx context.Context, category string, days *int) ([]contracts.LogStorageCleanupResult, error) {
	storage := GetLogStorageService(ctx)
	if storage == nil {
		return nil, errLogStorageUnavailable
	}
	var before *time.Time
	if days != nil {
		if *days <= 0 {
			return nil, fmt.Errorf("days must be greater than zero")
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -*days)
		before = &cutoff
	}
	return storage.Cleanup(ctx, category, before)
}

func GetLogStorageService(ctx context.Context) contracts.LogStorageService {
	if s, err := core.InjectFrom[contracts.LogStorageService](ctx); err == nil && s != nil {
		return s
	}
	servicesMu.RLock()
	defer servicesMu.RUnlock()
	return logStorageService
}
