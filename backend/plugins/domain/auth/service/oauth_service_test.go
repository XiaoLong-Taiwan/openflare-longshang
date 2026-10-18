// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package service

import (
	"Wavelet/core/contracts"
	"encoding/json"
	"testing"
)

func TestNormalizeOAuthUserInfoAcceptsStringProviderID(t *testing.T) {
	var userInfo contracts.OAuthUserInfoDTO
	if err := json.Unmarshal([]byte(`{"id":"casdoor-user-001","name":"Casdoor User"}`), &userInfo); err != nil {
		t.Fatalf("json.Unmarshal OAuthUserInfoDTO error = %v", err)
	}

	service := &OAuthService{}
	if err := service.NormalizeOAuthUserInfo(&userInfo); err != nil {
		t.Fatalf("NormalizeOAuthUserInfo(%+v) error = %v", userInfo, err)
	}

	if userInfo.Sub != "casdoor-user-001" {
		t.Errorf("NormalizeOAuthUserInfo(%+v) sub = %q, want %q", userInfo, userInfo.Sub, "casdoor-user-001")
	}
	if userInfo.Username != "casdoor-user-001" {
		t.Errorf("NormalizeOAuthUserInfo(%+v) username = %q, want %q", userInfo, userInfo.Username, "casdoor-user-001")
	}
}

func TestNormalizeOAuthUserInfoKeepsSubjectSeparateFromLocalUserID(t *testing.T) {
	userInfo := &contracts.OAuthUserInfoDTO{
		ID:       "external-string-id",
		Sub:      "oidc-subject",
		Username: "provider-user",
		Active:   true,
	}

	service := &OAuthService{}
	if err := service.NormalizeOAuthUserInfo(userInfo); err != nil {
		t.Fatalf("NormalizeOAuthUserInfo(%+v) error = %v", userInfo, err)
	}

	if userInfo.Sub != "oidc-subject" {
		t.Errorf("NormalizeOAuthUserInfo(%+v) sub = %q, want %q", userInfo, userInfo.Sub, "oidc-subject")
	}
}
