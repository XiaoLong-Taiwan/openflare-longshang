package driver_http

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

const trustedProxyConfigKey = "trusted_proxy_cidrs"

var cloudflareProxyCIDRs = []string{
	"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
	"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
	"197.234.240.0/22", "162.158.0.0/15", "104.16.0.0/13", "172.64.0.0/13",
	"131.0.72.0/22", "2400:cb00::/32", "2606:4700::/32", "2803:f800::/32",
	"2c0f:f248::/32", "2a06:98c0::/29",
}

type trustedProxyConfig struct {
	Enabled bool     `json:"enabled"`
	Mode    string   `json:"mode"`
	CIDRs   []string `json:"cidrs"`
}

func defaultTrustedProxyConfig() trustedProxyConfig {
	return trustedProxyConfig{Enabled: true, Mode: "append"}
}

func parseTrustedProxyConfig(raw string) ([]string, error) {
	cfg := defaultTrustedProxyConfig()
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
			return nil, fmt.Errorf("trusted proxy config: %w", err)
		}
	}
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.Mode != "append" && cfg.Mode != "override" {
		return nil, fmt.Errorf("trusted proxy mode must be append or override")
	}

	cidrs := cfg.CIDRs
	if cfg.Mode == "append" {
		cidrs = append(append([]string{}, cloudflareProxyCIDRs...), cidrs...)
	}
	result := make([]string, 0, len(cidrs))
	seen := make(map[string]struct{}, len(cidrs))
	for _, value := range cidrs {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, _, err := net.ParseCIDR(value); err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", value, err)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result, nil
}

func loadTrustedProxyCIDRs(ctx context.Context) (string, error) {
	db := getDB(ctx)
	if db == nil {
		return "", nil
	}
	var value string
	if err := db.Table("w_system_configs").Where("key = ?", trustedProxyConfigKey).Pluck("value", &value).Error; err != nil {
		return "", err
	}
	return value, nil
}

func configureTrustedProxies(engine *gin.Engine, ctx context.Context) error {
	raw, err := loadTrustedProxyCIDRs(ctx)
	if err != nil {
		return err
	}
	cidrs, err := parseTrustedProxyConfig(raw)
	if err != nil {
		return err
	}
	return engine.SetTrustedProxies(cidrs)
}
