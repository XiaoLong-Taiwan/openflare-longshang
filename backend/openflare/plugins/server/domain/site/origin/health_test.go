// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package origin

import "testing"

func TestIsHealthyStatus(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		accept4xx bool
		want      bool
	}{
		{name: "success", status: 200, want: true},
		{name: "redirect", status: 302, want: true},
		{name: "client error accepted", status: 400, accept4xx: true, want: true},
		{name: "client error rejected", status: 401, accept4xx: false, want: false},
		{name: "server error", status: 500, accept4xx: true, want: false},
		{name: "invalid status", status: 0, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHealthyStatus(tt.status, tt.accept4xx); got != tt.want {
				t.Errorf("isHealthyStatus(%d, %t) = %t, want %t", tt.status, tt.accept4xx, got, tt.want)
			}
		})
	}
}
