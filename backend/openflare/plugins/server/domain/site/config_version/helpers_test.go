// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package config_version

import "testing"

func TestSupportFilesEqual(t *testing.T) {
	tests := []struct {
		name     string
		previous string
		current  []SupportFile
		want     bool
	}{
		{
			name:     "same files",
			previous: `[{"path":"1.crt","content":"cert"}]`,
			current:  []SupportFile{{Path: "1.crt", Content: "cert"}},
			want:     true,
		},
		{
			name:     "changed content",
			previous: `[{"path":"1.crt","content":"old"}]`,
			current:  []SupportFile{{Path: "1.crt", Content: "new"}},
		},
		{
			name:     "invalid previous json",
			previous: `invalid`,
			current:  []SupportFile{{Path: "1.crt", Content: "cert"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := supportFilesEqual(test.previous, test.current); got != test.want {
				t.Fatalf("supportFilesEqual() = %v, want %v", got, test.want)
			}
		})
	}
}
