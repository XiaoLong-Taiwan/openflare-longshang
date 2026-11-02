// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package websocket manages persistent WebSocket connections between the OpenFlare server and its agents.
package websocket

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"Wavelet/openflare/plugins/server/domain/fleet/reporting"
	"Wavelet/openflare/share/protocol"
	"Wavelet/pkg/util"
)

const (
	// AgentWSConnectedLastSeenValue is the sentinel last_seen_at value when agent WS is connected.
	AgentWSConnectedLastSeenValue = "__OPENFLARE_WS_CONNECTED__"

	agentMessageTypeStatus          = "status"
	agentMessageTypeStatusAck       = "status_ack"
	agentMessageTypeSettings        = "settings"
	agentMessageTypeActiveConfig    = "active_config"
	agentMessageTypeForceSyncConfig = "force_sync_config"
	agentMessageTypeWAFIPGroups     = "waf_ip_groups"
)

// AgentStatusHandler processes inbound agent websocket status payloads.
type AgentStatusHandler func(ctx context.Context, nodeID, remoteAddr string, payload json.RawMessage) error

type agentClient struct {
	wsClientCore
	remoteAddr string
	onStatus   AgentStatusHandler
}

type agentHub struct {
	mu      sync.RWMutex
	clients map[string]*agentClient
}

var defaultAgentHub = &agentHub{clients: make(map[string]*agentClient)}

// ServeAgent handles an upgraded agent websocket connection.
func ServeAgent(c *gin.Context, nodeID string, onStatus AgentStatusHandler) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		slog.Debug("agent ws upgrade failed", "node_id", nodeID, "error", err)
		return
	}

	client := &agentClient{
		wsClientCore: wsClientCore{
			nodeID: nodeID,
			conn:   conn,
			send:   make(chan Message, wsChannelBuf),
			done:   make(chan struct{}),
		},
		remoteAddr: c.Request.RemoteAddr,
		onStatus:   onStatus,
	}
	defaultAgentHub.register(client)
	defer defaultAgentHub.unregister(client)

	slog.Debug("agent ws connected", "node_id", nodeID, "remote", client.remoteAddr)

	util.Go(client.writePump)
	client.readPump()
}

func (h *agentHub) register(client *agentClient) {
	h.mu.Lock()
	if existing := h.clients[client.nodeID]; existing != nil {
		existing.close()
	}
	h.clients[client.nodeID] = client
	h.mu.Unlock()
}

func (h *agentHub) unregister(client *agentClient) {
	h.mu.Lock()
	if current := h.clients[client.nodeID]; current == client {
		delete(h.clients, client.nodeID)
	}
	h.mu.Unlock()
	client.close()
}

// IsAgentConnected reports whether an agent websocket is active.
func IsAgentConnected(nodeID string) bool {
	defaultAgentHub.mu.RLock()
	client := defaultAgentHub.clients[nodeID]
	defaultAgentHub.mu.RUnlock()
	if client == nil {
		return false
	}
	select {
	case <-client.done:
		return false
	default:
		return true
	}
}

// SendAgentSettings pushes agent settings to a connected agent.
func SendAgentSettings(nodeID string, payload any) bool {
	return sendAgentMessage(nodeID, Message{Type: agentMessageTypeSettings, Payload: payload})
}

// SendAgentActiveConfig pushes active config metadata to a connected agent.
func SendAgentActiveConfig(nodeID string, payload any) bool {
	return sendAgentMessage(nodeID, Message{Type: agentMessageTypeActiveConfig, Payload: payload})
}

// SendAgentWAFIPGroups pushes WAF IP group updates to a connected agent.
func SendAgentWAFIPGroups(nodeID string, payload any) bool {
	return sendAgentMessage(nodeID, Message{Type: agentMessageTypeWAFIPGroups, Payload: payload})
}

// BroadcastWAFIPGroups pushes changed WAF IP groups to all connected agents.
func BroadcastWAFIPGroups(payload any) int {
	return broadcastAgent(agentMessageTypeWAFIPGroups, payload)
}

// BroadcastActiveConfig pushes active config metadata to all connected agents.
func BroadcastActiveConfig(payload any) int {
	return broadcastAgent(agentMessageTypeActiveConfig, payload)
}

func broadcastAgent(messageType string, payload any) int {
	if payload == nil {
		return 0
	}
	message := Message{Type: messageType, Payload: payload}
	defaultAgentHub.mu.RLock()
	clients := make([]*agentClient, 0, len(defaultAgentHub.clients))
	for _, client := range defaultAgentHub.clients {
		clients = append(clients, client)
	}
	defaultAgentHub.mu.RUnlock()

	success := 0
	for _, client := range clients {
		if client.enqueue(message) {
			success++
		}
	}
	return success
}

// SendForceSyncConfig notifies an agent to force sync configuration.
func SendForceSyncConfig(nodeID string, payload any) bool {
	return sendAgentMessage(nodeID, Message{Type: agentMessageTypeForceSyncConfig, Payload: payload})
}

func sendAgentMessage(nodeID string, message Message) bool {
	defaultAgentHub.mu.RLock()
	client := defaultAgentHub.clients[nodeID]
	defaultAgentHub.mu.RUnlock()
	if client == nil {
		return false
	}
	return client.enqueue(message)
}

func (c *agentClient) readPump() {
	defer c.close()
	c.conn.SetReadLimit(reporting.MaxPayloadBytes)

	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(agentWSReadTimeout()))
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			slog.Debug("agent ws read closed", "node_id", c.nodeID, "error", err)
			return
		}

		var inbound struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload,omitempty"`
		}
		if err = json.Unmarshal(data, &inbound); err != nil {
			slog.Debug("agent ws invalid message", "node_id", c.nodeID, "error", err)
			continue
		}

		slog.Debug("agent ws message received", "node_id", c.nodeID, "type", inbound.Type)
		switch inbound.Type {
		case agentMessageTypeStatus:
			if !validStatusPayload(inbound.Payload) {
				_ = c.enqueue(Message{Type: agentMessageTypeStatusAck, Payload: statusAck{Accepted: false}})
				continue
			}
			if c.onStatus != nil {
				if err := c.onStatus(context.Background(), c.nodeID, c.remoteAddr, inbound.Payload); err != nil {
					_ = c.enqueue(Message{Type: agentMessageTypeStatusAck, Payload: statusAck{Accepted: false}})
					continue
				}
			}
			_ = c.enqueue(Message{Type: agentMessageTypeStatusAck, Payload: statusAck{Accepted: true}})
		case messageTypePing:
			_ = c.enqueue(Message{Type: messageTypePong})
		case messageTypePong:
		default:
			slog.Debug("agent ws unsupported message type", "node_id", c.nodeID, "type", inbound.Type)
		}
	}
}

type statusAck struct {
	Accepted bool `json:"accepted"`
}

func validStatusPayload(raw []byte) bool {
	var payload protocol.NodePayload
	return json.Unmarshal(raw, &payload) == nil && reporting.Valid(payload)
}

func agentWSReadTimeout() time.Duration {
	timeout := wsReadDeadline
	if timeout < minAgentWSReadTimeout {
		return minAgentWSReadTimeout
	}
	return timeout
}

func (c *agentClient) writePump() {
	runWritePump(c.nodeID, c.conn, c.done, c.send, c.close, "agent ws")
}
