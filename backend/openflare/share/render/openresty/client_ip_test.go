// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package openresty

import (
	"strings"
	"testing"
)

func TestDefaultOpenRestyMainConfigIncludesCloudflareClientIP(t *testing.T) {
	config := RenderMainConfig(Document{})
	if !strings.Contains(config, `"client_ip":"$http_cf_connecting_ip"`) {
		t.Fatal("expected access log template to include CF-Connecting-IP")
	}
}
