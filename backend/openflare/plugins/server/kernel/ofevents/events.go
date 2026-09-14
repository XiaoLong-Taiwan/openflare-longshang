// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package ofevents lists OpenFlare-specific push events registered onto Wavelet's PushRegistry.
package ofevents

import "Wavelet/core/contracts"

// All returns OpenFlare-only built-in push events.
// Platform events (admin login, etc.) stay in Wavelet message_gateway.
func All() []contracts.PushEventMeta {
	return nil
}
