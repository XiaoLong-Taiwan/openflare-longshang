// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package websocket

import (
	"sync"
	"testing"
)

func TestWSClientCoreCloseIsIdempotent(t *testing.T) {
	core := &wsClientCore{
		send: make(chan Message, 1),
		done: make(chan struct{}),
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			core.close()
		}()
	}
	wg.Wait()
	select {
	case <-core.done:
	default:
		t.Fatal("close did not signal done")
	}
}

func TestWSClientCoreEnqueueFailsAfterClose(t *testing.T) {
	core := &wsClientCore{
		send: make(chan Message, 1),
		done: make(chan struct{}),
	}
	core.close()
	// 循环多次：若 close 检查与发送合并在同一个 select，两 case 同时就绪时
	// Go 随机选择，单次调用可能碰巧通过。
	for range 50 {
		if core.enqueue(Message{Type: messageTypePing}) {
			t.Fatal("enqueue must fail after close")
		}
	}
}

func TestWSClientCoreEnqueueNeverBlocks(t *testing.T) {
	core := &wsClientCore{
		send: make(chan Message, 1), // 缓冲小于消息数，验证不阻塞
		done: make(chan struct{}),
	}
	defer core.close()
	for range 3 {
		if !core.enqueue(Message{Type: messageTypePing}) && len(core.send) == 0 {
			t.Fatal("enqueue failed with empty buffer")
		}
	}
	if core.enqueue(Message{Type: messageTypePing}) {
		t.Fatal("enqueue must fail when buffer full")
	}
}
