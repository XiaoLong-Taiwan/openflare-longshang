// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package proxy_route

import "testing"

func TestNormalizeCacheConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   CacheConfigInput
		wantErr bool
	}{
		{name: "valid", input: CacheConfigInput{SuccessTTL: " 2h ", RedirectTTL: "30m", NotFoundTTL: "1m", BypassCookies: []string{"session_id", "session_id"}}},
		{name: "invalid ttl", input: CacheConfigInput{SuccessTTL: "2h; include bad"}, wantErr: true},
		{name: "invalid cookie", input: CacheConfigInput{BypassCookies: []string{"session-id"}}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeCacheConfig(true, test.input)
			if (err != nil) != test.wantErr {
				t.Fatalf("normalizeCacheConfig(%#v) error = %v, want error presence = %t", test.input, err, test.wantErr)
			}
			if err == nil && test.name == "valid" {
				if got.SuccessTTL != "2h" || len(got.BypassCookies) != 1 {
					t.Errorf("normalizeCacheConfig(%#v) = %#v, want trimmed TTL and deduplicated cookies", test.input, got)
				}
			}
		})
	}
}
