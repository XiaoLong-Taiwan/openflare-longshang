// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

// Package apply_log manages the application of configuration change logs,
// including validation and retention policy enforcement.
package apply_log

const (
	errRetentionDaysOutOfRange = "retention_days 必须在 1 到 3650 之间"
)
