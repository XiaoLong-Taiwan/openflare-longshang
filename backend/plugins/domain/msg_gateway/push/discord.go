// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package push

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"Wavelet/pkg/httppool"
)

func init() {
	Register("discord", &DiscordPusher{})
}

// DiscordPusher 使用 Discord Incoming Webhook 发送推送。
type DiscordPusher struct{}

// Send 发送 Discord webhook 通知。
func (p *DiscordPusher) Send(ctx context.Context, cfg Config, _ string, body map[string]any, template string, _ map[string]any) (string, error) {
	if cfg.URL == "" {
		return "", errors.New("discord: webhook URL is required")
	}

	payload := template
	if payload == "" {
		payload = cfg.Other
	}
	if payload != "" {
		payload = ParseTemplate(payload, body)
	} else {
		payloadBytes, err := json.Marshal(map[string]any{
			"content": bodyContent(body, "**%s**: %v", "\n"),
		})
		if err != nil {
			return "", fmt.Errorf("discord: marshal payload failed: %w", err)
		}
		payload = string(payloadBytes)
	}
	if !json.Valid([]byte(payload)) {
		return "", errors.New("discord: payload must be valid JSON")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader([]byte(payload)))
	if err != nil {
		return "", fmt.Errorf("discord: create http request failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httppool.NewClient(defaultHTTPClientTimeout).Do(req)
	if err != nil {
		return "", fmt.Errorf("discord: webhook request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes))
	upstreamResp := strings.TrimSpace(string(responseBody))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamResp, fmt.Errorf("discord: webhook returned %s", resp.Status)
	}
	return upstreamResp, nil
}

// ValidateConfig 校验 Discord Webhook 配置。
func (p *DiscordPusher) ValidateConfig(cfg Config) error {
	if cfg.URL == "" {
		return errors.New("webhook URL is required")
	}
	if !strings.HasPrefix(cfg.URL, "https://") {
		return errors.New("webhook URL must use https:// protocol")
	}
	if cfg.Other != "" && !json.Valid([]byte(cfg.Other)) {
		return errors.New("payload must be valid JSON")
	}
	return nil
}
