// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package websocket

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

// runWritePump drains send onto conn until done is closed, emitting
// JSON pings at wsPingInterval. Shared by agent/relay/flared clients;
// closeFn must be idempotent.
func runWritePump(
	nodeID string,
	conn *websocket.Conn,
	done <-chan struct{},
	send chan Message,
	closeFn func(),
	logLabel string,
) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case message := <-send:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteDeadline))
			if err := conn.WriteJSON(message); err != nil {
				slog.Debug(logLabel+" write failed", "node_id", nodeID, "error", err)
				closeFn()
				return
			}
		case <-ticker.C:
			select {
			case <-done:
				return
			case send <- Message{Type: messageTypePing}:
			default:
			}
		}
	}
}
