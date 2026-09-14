// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package apiutil

import "Wavelet/core/contracts"

// AdminMiddlewares returns Wavelet-standard middlewares for OpenFlare console routes.
// OpenFlare no longer distinguishes Admin vs Root tiers; all management endpoints share
// the same gate: RequireAuth + RequireAdmin from the platform AuthService.
//
// 返回 []any 而非 []gin.HandlerFunc：内核 RouterExtension.Use 收 ...any，
// 而 Go 不允许把 []T 直接展开成 ...any。
func AdminMiddlewares(auth contracts.AuthService) []any {
	return []any{auth.RequireAuthMiddleware(), auth.RequireAdminMiddleware()}
}
