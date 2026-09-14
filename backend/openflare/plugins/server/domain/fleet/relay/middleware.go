// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package relay

import (
	"strings"

	"Wavelet/openflare/plugins/server/domain/fleet/agent"

	"Wavelet/pkg/response"

	"github.com/gin-gonic/gin"
)

const ctxRelayNodeKey = "relay_node"

// Auth authenticates relay requests using X-Agent-Token and verifies tunnel_relay type.
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := strings.TrimSpace(c.GetHeader("X-Agent-Token"))
		node, err := agent.AuthenticateAccessToken(c.Request.Context(), token)
		if err != nil {
			response.AbortUnauthorized(c, errAgentTokenInvalid)
			return
		}
		if node.NodeType != "tunnel_relay" {
			response.AbortForbidden(c, errRelayNodeTypeMismatch)
			return
		}
		c.Set(ctxRelayNodeKey, node)
		c.Next()
	}
}
