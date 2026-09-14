// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package flared defines shared error messages for tunnel client operations.
package flared

const (
	errTunnelTokenInvalid     = "无权进行此操作，Tunnel Token 无效" //nolint:gosec // false positive: this is an error message, not hardcoded credentials
	errTunnelNodeTypeMismatch = "此节点不是 TunnelClient 类型"
)
