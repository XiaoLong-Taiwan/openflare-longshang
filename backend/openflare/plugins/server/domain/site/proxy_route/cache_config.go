// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package proxy_route

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
)

var (
	cacheTTLPattern    = regexp.MustCompile(`^(?:unlimited|[1-9][0-9]*(?:s|m|h|d|w))$`)
	cacheCookiePattern = regexp.MustCompile(`^[A-Za-z0-9_]{1,64}$`)
)

// CacheConfigInput defines validated per-route Nginx cache behavior.
type CacheConfigInput struct {
	SuccessTTL          string   `json:"success_ttl,omitempty"`
	RedirectTTL         string   `json:"redirect_ttl,omitempty"`
	NotFoundTTL         string   `json:"not_found_ttl,omitempty"`
	BypassAuthorization bool     `json:"bypass_authorization,omitempty"`
	BypassCookies       []string `json:"bypass_cookies,omitempty"`
}

func normalizeCacheConfig(enabled bool, input CacheConfigInput) (CacheConfigInput, error) {
	if !enabled {
		return CacheConfigInput{}, nil
	}
	input.SuccessTTL = strings.TrimSpace(input.SuccessTTL)
	input.RedirectTTL = strings.TrimSpace(input.RedirectTTL)
	input.NotFoundTTL = strings.TrimSpace(input.NotFoundTTL)
	for _, ttl := range []string{input.SuccessTTL, input.RedirectTTL, input.NotFoundTTL} {
		if ttl != "" && !cacheTTLPattern.MatchString(ttl) {
			return CacheConfigInput{}, errors.New(errProxyRouteCacheTTL)
		}
	}
	seen := make(map[string]struct{}, len(input.BypassCookies))
	cookies := make([]string, 0, len(input.BypassCookies))
	for _, raw := range input.BypassCookies {
		cookie := strings.TrimSpace(raw)
		if !cacheCookiePattern.MatchString(cookie) {
			return CacheConfigInput{}, errors.New(errProxyRouteCacheCookie)
		}
		if _, exists := seen[cookie]; exists {
			continue
		}
		seen[cookie] = struct{}{}
		cookies = append(cookies, cookie)
	}
	input.BypassCookies = cookies
	return input, nil
}

// DecodeStoredCacheConfig decodes a persisted per-route cache configuration.
func DecodeStoredCacheConfig(raw string) (CacheConfigInput, error) {
	if strings.TrimSpace(raw) == "" {
		return CacheConfigInput{}, nil
	}
	var config CacheConfigInput
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		return CacheConfigInput{}, err
	}
	return normalizeCacheConfig(true, config)
}
