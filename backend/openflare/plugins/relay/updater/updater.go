// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package updater provides relay self-update integration with the edge updater.
package updater

import (
	"Wavelet/openflare/plugins/relay/config"
	edgeupdater "Wavelet/openflare/share/edge/updater"
)

// Service is an alias for the edge updater Service.
type Service = edgeupdater.Service

// UpdateOptions is an alias for the edge updater UpdateOptions.
type UpdateOptions = edgeupdater.UpdateOptions

// New creates and initializes a new updater Service for the relay.
func New() *Service {
	return edgeupdater.New(edgeupdater.Config{
		LocalVersion: config.Version,
		AssetPrefix:  "openflare-relay",
		LogLabel:     "relay",
	})
}
