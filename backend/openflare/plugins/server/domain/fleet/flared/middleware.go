// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package flared

import (
	"strings"

	"Wavelet/openflare/plugins/server/domain/fleet/agent"

	"Wavelet/pkg/response"

	"github.com/gin-gonic/gin"
)

const ctxFlaredNodeKey = "flared_node"

// TunnelAuth authenticates flared requests using X-Tunnel-Token and verifies tunnel_client type.
func TunnelAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("X-Tunnel-Token"))
		node, err := agent.AuthenticateAccessToken(c.Request.Context(), token)
		if err != nil {
			response.AbortUnauthorized(c, errTunnelTokenInvalid)
			return
		}
		if node.NodeType != "tunnel_client" {
			response.AbortForbidden(c, errTunnelNodeTypeMismatch)
			return
		}
		c.Set(ctxFlaredNodeKey, node)
		c.Next()
	}
}
