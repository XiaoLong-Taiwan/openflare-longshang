// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package websocket

import "time"

const (
	wsChannelBuf          = 16
	wsPingInterval        = 30 * time.Second
	wsReadDeadline        = 90 * time.Second
	wsWriteDeadline       = 10 * time.Second
	minAgentWSReadTimeout = 30 * time.Second
)
