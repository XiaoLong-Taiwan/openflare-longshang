// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// wsClientCore holds the state and lifecycle shared by all WebSocket client
// variants (agent/relay/flared). Embed it; call close exactly-once semantics
// are guaranteed via once.
type wsClientCore struct {
	nodeID string
	conn   *websocket.Conn
	send   chan Message
	done   chan struct{}
	once   sync.Once
}

// close tears down the connection at most once.
func (c *wsClientCore) close() {
	if c == nil {
		return
	}
	c.once.Do(func() {
		close(c.done)
		if c.conn != nil {
			_ = c.conn.Close()
		}
	})
}

// enqueue best-effort delivers message; it never blocks and fails fast when
// the client is closed or its send buffer is full.
func (c *wsClientCore) enqueue(message Message) bool {
	// 先确定性检查 closed：若与发送合并在同一个 select，两个 case 同时就绪时
	// Go 会随机选择，close 后仍可能投递成功。
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.send <- message:
		return true
	default:
		return false
	}
}
