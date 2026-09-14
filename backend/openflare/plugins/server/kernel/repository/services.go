// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package repository

import (
	"sync"

	"Wavelet/core/contracts"
)

var (
	svcMu   sync.RWMutex
	authSvc contracts.AuthService
	userSvc contracts.UserService
)

// SetAuthService injects the platform AuthService used by GetActiveAuthSources.
func SetAuthService(s contracts.AuthService) {
	svcMu.Lock()
	defer svcMu.Unlock()
	authSvc = s
}

// SetUserService injects the platform UserService used by GetSystemUser.
func SetUserService(s contracts.UserService) {
	svcMu.Lock()
	defer svcMu.Unlock()
	userSvc = s
}

func currentAuthService() contracts.AuthService {
	svcMu.RLock()
	defer svcMu.RUnlock()
	return authSvc
}

func currentUserService() contracts.UserService {
	svcMu.RLock()
	defer svcMu.RUnlock()
	return userSvc
}
