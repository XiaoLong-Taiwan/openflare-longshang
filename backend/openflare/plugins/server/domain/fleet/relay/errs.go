// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package relay provides relay node management and authentication for the OpenFlare platform.
package relay

const (
	//nolint:gosec // error message text, not a credential
	errAgentTokenInvalid     = "无权进行此操作，Agent Token 无效"
	errRelayNodeTypeMismatch = "此节点不是 TunnelRelay 类型"
)
