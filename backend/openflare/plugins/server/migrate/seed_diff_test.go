// Copyright 2026 Arctel.net
// SPDX-License-Identifier: Apache-2.0

package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var ofScheduleSeeds = []string{
	"of_ssl_renew",
	"of_waf_ip_group_sync",
	"of_uptime_kuma_sync",
	"of_pages_source_scan",
}

// ofConfigSeedKeys is G \ W from a gold v3.5.4 empty-DB goose run minus Wavelet admin 00001 keys.
var ofConfigSeedKeys = []string{
	"agent_heartbeat_interval",
	"agent_update_repo",
	"agent_websocket_upgrade_enabled",
	"geoip_provider",
	"log_retention_days_clickhouse",
	"log_retention_days_postgres",
	"log_retention_days_sqlite",
	"metric_retention_days",
	"node_offline_threshold",
	"openresty_cache_enabled",
	"openresty_cache_inactive",
	"openresty_cache_key_template",
	"openresty_cache_levels",
	"openresty_cache_lock_enabled",
	"openresty_cache_lock_timeout",
	"openresty_cache_max_size",
	"openresty_cache_use_stale",
	"openresty_client_body_timeout",
	"openresty_client_header_timeout",
	"openresty_client_max_body_size",
	"openresty_default_limit_conn_per_ip",
	"openresty_default_limit_conn_per_server",
	"openresty_default_limit_rate",
	"openresty_default_limit_req_per_ip",
	"openresty_default_server_return_status",
	"openresty_events_multi_accept_enabled",
	"openresty_events_use",
	"openresty_gzip_comp_level",
	"openresty_gzip_enabled",
	"openresty_gzip_min_length",
	"openresty_http3_enabled",
	"openresty_keepalive_requests",
	"openresty_keepalive_timeout",
	"openresty_large_client_header_buffers",
	"openresty_proxy_buffer_size",
	"openresty_proxy_buffering_enabled",
	"openresty_proxy_buffers",
	"openresty_proxy_busy_buffers_size",
	"openresty_proxy_connect_timeout",
	"openresty_proxy_read_timeout",
	"openresty_proxy_request_buffering_enabled",
	"openresty_proxy_send_timeout",
	"openresty_send_timeout",
	"openresty_websocket_enabled",
	"openresty_worker_connections",
	"openresty_worker_processes",
	"openresty_worker_rlimit_nofile",
	"origin_error_page_enabled",
	"origin_error_page_get_only",
	"origin_error_page_html",
	"origin_error_page_status_codes",
	"pages_max_history_count",
	"pages_max_package_size_mb",
	"relay_frps_web_ui_enabled",
	"relay_frps_web_ui_port",
	"sw_offline_domains",
	"sw_offline_enabled",
	"sw_offline_html",
	"uptime_kuma_enabled",
	"uptime_kuma_interval",
	"uptime_kuma_monitor_scope",
	"uptime_kuma_retry",
	"uptime_kuma_retry_interval",
	"uptime_kuma_sync_interval",
	"uptime_kuma_timeout",
}

var waveletOwnedKeys = []string{
	"cap_login_enabled",
	"smtp_host",
	"storage_config",
	"site_name",
}

func TestInitialSQLContainsScheduleSeeds(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgres"} {
		s := readInitialSQL(t, dialect)
		for _, needle := range ofScheduleSeeds {
			if !strings.Contains(s, needle) {
				t.Errorf("%s 00001 missing schedule seed %s", dialect, needle)
			}
		}
		if strings.Contains(s, "of_database_auto_cleanup") {
			t.Errorf("%s 00001 must not seed of_database_auto_cleanup", dialect)
		}
	}
}

func TestInitialSQLContainsConfigSeedDiff(t *testing.T) {
	for _, dialect := range []string{"sqlite", "postgres"} {
		s := readInitialSQL(t, dialect)
		if !strings.Contains(s, "INSERT") {
			t.Errorf("%s 00001 has no INSERT", dialect)
		}
		for _, key := range ofConfigSeedKeys {
			if !strings.Contains(s, "'"+key+"'") {
				t.Errorf("%s 00001 missing config seed %s", dialect, key)
			}
		}
		for _, key := range waveletOwnedKeys {
			if strings.Contains(s, "'"+key+"'") {
				t.Errorf("%s 00001 must not seed Wavelet key %s", dialect, key)
			}
		}
	}
}

func readInitialSQL(t *testing.T, dialect string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dialect, "00001_initial.sql"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
