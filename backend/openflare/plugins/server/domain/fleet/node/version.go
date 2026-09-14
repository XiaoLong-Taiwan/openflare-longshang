// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package node

import "Wavelet/openflare/share/ofutil"

func compareVersions(local, remote string) int {
	return ofutil.CompareVersions(local, remote)
}
