// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package logging configures structured logging for the agent process.
package logging

import edgelogging "Wavelet/openflare/share/edge/logging"

// Setup initialises structured logging for the agent process.
func Setup() {
	edgelogging.Setup(edgelogging.Options{AddSource: true})
}
