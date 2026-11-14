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

func TestClassifyHealthStatus(t *testing.T) {
	tests := []struct {
		status int
		online bool
		want   string
	}{
		{status: 200, online: true, want: "healthy"},
		{status: 404, online: false, want: "http_4xx"},
		{status: 503, online: false, want: "http_5xx"},
		{status: 302, online: false, want: "unexpected_status"},
	}
	for _, test := range tests {
		if got := classifyHealthStatus(test.status, test.online); got != test.want {
			t.Errorf("classifyHealthStatus(%d, %t) = %q, want %q", test.status, test.online, got, test.want)
		}
	}
}
