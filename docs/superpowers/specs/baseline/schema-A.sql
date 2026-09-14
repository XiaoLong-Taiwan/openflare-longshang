-- index idx_of_apply_logs_created_at
CREATE INDEX idx_of_apply_logs_created_at ON of_apply_logs(created_at);
-- index idx_of_apply_logs_node_id
CREATE INDEX idx_of_apply_logs_node_id ON of_apply_logs(node_id);
-- index idx_of_cf_connections_dns_account_id
CREATE INDEX idx_of_cf_connections_dns_account_id ON of_cf_connections (dns_account_id);
-- index idx_of_cf_pointing_groups_active_node_id
CREATE INDEX idx_of_cf_pointing_groups_active_node_id ON of_cf_pointing_groups (active_node_id);
-- index idx_of_cf_pointing_groups_backup_node_id
CREATE INDEX idx_of_cf_pointing_groups_backup_node_id ON of_cf_pointing_groups (backup_node_id);
-- index idx_of_cf_pointing_groups_primary_node_id
CREATE INDEX idx_of_cf_pointing_groups_primary_node_id ON of_cf_pointing_groups (primary_node_id);
-- index idx_of_cf_pointing_members_group_id
CREATE INDEX idx_of_cf_pointing_members_group_id ON of_cf_pointing_members (group_id);
-- index idx_of_cf_pointing_members_zone_domain_id
CREATE UNIQUE INDEX idx_of_cf_pointing_members_zone_domain_id ON of_cf_pointing_members (zone_domain_id);
-- index idx_of_config_versions_is_active
CREATE INDEX idx_of_config_versions_is_active ON of_config_versions (is_active);
-- index idx_of_node_access_logs_host
CREATE INDEX idx_of_node_access_logs_host ON of_node_access_logs (host, logged_at DESC);
-- index idx_of_node_access_logs_host_lower
CREATE INDEX idx_of_node_access_logs_host_lower ON of_node_access_logs (lower(trim(host)));
-- index idx_of_node_access_logs_logged_at
CREATE INDEX idx_of_node_access_logs_logged_at ON of_node_access_logs (logged_at DESC, id DESC);
-- index idx_of_node_access_logs_node_id
CREATE INDEX idx_of_node_access_logs_node_id ON of_node_access_logs (node_id, logged_at DESC);
-- index idx_of_node_access_logs_remote_addr
CREATE INDEX idx_of_node_access_logs_remote_addr ON of_node_access_logs (remote_addr, logged_at DESC);
-- index idx_of_node_access_logs_status_code
CREATE INDEX idx_of_node_access_logs_status_code ON of_node_access_logs (status_code, logged_at DESC);
-- index idx_of_node_edge_health_node
CREATE INDEX idx_of_node_edge_health_node ON of_node_edge_health (node_id, captured_at DESC);
-- index idx_of_node_health_events_event_type
CREATE INDEX idx_of_node_health_events_event_type ON of_node_health_events (event_type);
-- index idx_of_node_health_events_first_triggered_at
CREATE INDEX idx_of_node_health_events_first_triggered_at ON of_node_health_events (first_triggered_at);
-- index idx_of_node_health_events_last_triggered_at
CREATE INDEX idx_of_node_health_events_last_triggered_at ON of_node_health_events (last_triggered_at);
-- index idx_of_node_health_events_node_id
CREATE INDEX idx_of_node_health_events_node_id ON of_node_health_events (node_id);
-- index idx_of_node_health_events_reported_at
CREATE INDEX idx_of_node_health_events_reported_at ON of_node_health_events (reported_at);
-- index idx_of_node_health_events_resolved_at
CREATE INDEX idx_of_node_health_events_resolved_at ON of_node_health_events (resolved_at);
-- index idx_of_node_health_events_status
CREATE INDEX idx_of_node_health_events_status ON of_node_health_events (status);
-- index idx_of_node_metric_snapshots_node
CREATE INDEX idx_of_node_metric_snapshots_node ON of_node_metric_snapshots (node_id, captured_at DESC);
-- index idx_of_node_obs_frpc_node
CREATE INDEX idx_of_node_obs_frpc_node ON of_node_obs_frpc (node_id, captured_at DESC);
-- index idx_of_node_obs_frps_node
CREATE INDEX idx_of_node_obs_frps_node ON of_node_obs_frps (node_id, captured_at DESC);
-- index idx_of_node_system_profiles_node_id
CREATE UNIQUE INDEX idx_of_node_system_profiles_node_id ON of_node_system_profiles (node_id);
-- index idx_of_node_system_profiles_reported_at
CREATE INDEX idx_of_node_system_profiles_reported_at ON of_node_system_profiles (reported_at);
-- index idx_of_nodes_access_token
CREATE INDEX idx_of_nodes_access_token ON of_nodes (access_token);
-- index idx_of_nodes_node_id
CREATE UNIQUE INDEX idx_of_nodes_node_id ON of_nodes (node_id);
-- index idx_of_origins_address
CREATE UNIQUE INDEX idx_of_origins_address ON of_origins (address);
-- index idx_of_pages_deployment_files_deployment_id
CREATE INDEX idx_of_pages_deployment_files_deployment_id ON of_pages_deployment_files (deployment_id);
-- index idx_of_pages_deployments_checksum
CREATE INDEX idx_of_pages_deployments_checksum ON of_pages_deployments (checksum);
-- index idx_of_pages_deployments_project_id
CREATE INDEX idx_of_pages_deployments_project_id ON of_pages_deployments (project_id);
-- index idx_of_pages_deployments_project_number
CREATE UNIQUE INDEX idx_of_pages_deployments_project_number
    ON of_pages_deployments (project_id, deployment_number);
-- index idx_of_pages_deployments_source_revision
CREATE UNIQUE INDEX idx_of_pages_deployments_source_revision
    ON of_pages_deployments (project_id, source_identity, source_revision)
    WHERE source_identity IS NOT NULL AND source_revision IS NOT NULL;
-- index idx_of_pages_deployments_status
CREATE INDEX idx_of_pages_deployments_status ON of_pages_deployments (status);
-- index idx_of_pages_deployments_upload_id
CREATE INDEX idx_of_pages_deployments_upload_id ON of_pages_deployments (upload_id);
-- index idx_of_pages_project_source_runtime_next_check_at
CREATE INDEX idx_of_pages_project_source_runtime_next_check_at
    ON of_pages_project_source_runtime (next_check_at);
-- index idx_of_pages_project_sources_project_id
CREATE UNIQUE INDEX idx_of_pages_project_sources_project_id
    ON of_pages_project_sources (project_id);
-- index idx_of_pages_projects_active_deployment_id
CREATE INDEX idx_of_pages_projects_active_deployment_id ON of_pages_projects (active_deployment_id);
-- index idx_of_pages_projects_slug
CREATE UNIQUE INDEX idx_of_pages_projects_slug ON of_pages_projects (slug);
-- index idx_of_proxy_routes_origin_id
CREATE INDEX idx_of_proxy_routes_origin_id ON of_proxy_routes (origin_id);
-- index idx_of_proxy_routes_pages_project_id
CREATE INDEX idx_of_proxy_routes_pages_project_id ON of_proxy_routes (pages_project_id);
-- index idx_of_proxy_routes_site_name
CREATE UNIQUE INDEX idx_of_proxy_routes_site_name ON of_proxy_routes (site_name);
-- index idx_of_proxy_routes_tunnel_node_id
CREATE INDEX idx_of_proxy_routes_tunnel_node_id ON of_proxy_routes (tunnel_node_id);
-- index idx_of_tls_certificates_name
CREATE UNIQUE INDEX idx_of_tls_certificates_name ON of_tls_certificates (name);
-- index idx_of_waf_group_route
CREATE UNIQUE INDEX idx_of_waf_group_route ON of_waf_rule_group_bindings (rule_group_id, proxy_route_id);
-- index idx_of_waf_ip_groups_next_sync_at
CREATE INDEX idx_of_waf_ip_groups_next_sync_at ON of_waf_ip_groups (next_sync_at);
-- index idx_of_waf_ip_groups_type
CREATE INDEX idx_of_waf_ip_groups_type ON of_waf_ip_groups (type);
-- index idx_of_waf_rule_group_bindings_proxy_route_id
CREATE INDEX idx_of_waf_rule_group_bindings_proxy_route_id ON of_waf_rule_group_bindings (proxy_route_id);
-- index idx_of_waf_rule_groups_is_global
CREATE INDEX idx_of_waf_rule_groups_is_global ON of_waf_rule_groups (is_global);
-- index idx_of_zone_domains_cert_id
CREATE INDEX idx_of_zone_domains_cert_id ON of_zone_domains (cert_id);
-- index idx_of_zone_domains_domain
CREATE UNIQUE INDEX idx_of_zone_domains_domain ON of_zone_domains (domain);
-- index idx_of_zone_domains_proxy_route_id
CREATE INDEX idx_of_zone_domains_proxy_route_id ON of_zone_domains (proxy_route_id);
-- index idx_of_zone_domains_zone_id
CREATE INDEX idx_of_zone_domains_zone_id ON of_zone_domains (zone_id);
-- index idx_of_zones_domain
CREATE UNIQUE INDEX idx_of_zones_domain ON of_zones (domain);
-- index idx_w_access_tokens_user_id
CREATE INDEX idx_w_access_tokens_user_id ON w_access_tokens (user_id);
-- index idx_w_auth_sources_is_active
CREATE INDEX idx_w_auth_sources_is_active ON w_auth_sources (is_active);
-- index idx_w_external_accounts_auth_source_id
CREATE INDEX idx_w_external_accounts_auth_source_id ON w_external_accounts (auth_source_id);
-- index idx_w_external_accounts_source_external
CREATE UNIQUE INDEX idx_w_external_accounts_source_external ON w_external_accounts (auth_source_id, external_id);
-- index idx_w_external_accounts_user_id
CREATE INDEX idx_w_external_accounts_user_id ON w_external_accounts (user_id);
-- index idx_w_push_channels_enabled
CREATE INDEX idx_w_push_channels_enabled ON w_push_channels(enabled);
-- index idx_w_push_channels_name
CREATE INDEX idx_w_push_channels_name ON w_push_channels(name);
-- index idx_w_push_events_enabled
CREATE INDEX idx_w_push_events_enabled ON w_push_events(enabled);
-- index idx_w_push_events_task_type
CREATE INDEX idx_w_push_events_task_type ON w_push_events(task_type);
-- index idx_w_push_histories_created
CREATE INDEX idx_w_push_histories_created ON w_push_histories(created_at);
-- index idx_w_push_histories_event
CREATE INDEX idx_w_push_histories_event ON w_push_histories(event_key);
-- index idx_w_schedules_is_active
CREATE INDEX idx_w_schedules_is_active ON w_schedules (is_active);
-- index idx_w_task_executions_created_at
CREATE INDEX idx_w_task_executions_created_at ON w_task_executions (created_at);
-- index idx_w_task_executions_started_at
CREATE INDEX idx_w_task_executions_started_at ON w_task_executions (started_at);
-- index idx_w_task_executions_status
CREATE INDEX idx_w_task_executions_status ON w_task_executions (status);
-- index idx_w_task_executions_task_type
CREATE INDEX idx_w_task_executions_task_type ON w_task_executions (task_type);
-- index idx_w_templates_created_at
CREATE INDEX idx_w_templates_created_at ON w_templates (created_at);
-- index idx_w_templates_is_system
CREATE INDEX idx_w_templates_is_system ON w_templates (is_system);
-- index idx_w_templates_updated_at
CREATE INDEX idx_w_templates_updated_at ON w_templates (updated_at);
-- index idx_w_uploads_file_path
CREATE INDEX idx_w_uploads_file_path ON w_uploads (file_path);
-- index idx_w_uploads_hash
CREATE INDEX idx_w_uploads_hash ON w_uploads (hash);
-- index idx_w_uploads_hash_file_size_status
CREATE INDEX idx_w_uploads_hash_file_size_status ON w_uploads (hash, file_size, status);
-- index idx_w_uploads_status_created_at
CREATE INDEX idx_w_uploads_status_created_at ON w_uploads (status, created_at);
-- index idx_w_uploads_type
CREATE INDEX idx_w_uploads_type ON w_uploads (type);
-- index idx_w_uploads_user_id
CREATE INDEX idx_w_uploads_user_id ON w_uploads (user_id);
-- index idx_w_user_access_logs_user_id
CREATE INDEX idx_w_user_access_logs_user_id ON w_user_access_logs (user_id, created_at DESC);
-- index idx_w_users_created_at
CREATE INDEX idx_w_users_created_at ON w_users (created_at);
-- index idx_w_users_email
CREATE INDEX idx_w_users_email ON w_users (email);
-- index idx_w_users_is_active
CREATE INDEX idx_w_users_is_active ON w_users (is_active);
-- index idx_w_users_last_login_at
CREATE INDEX idx_w_users_last_login_at ON w_users (last_login_at);
-- table of_acme_accounts
CREATE TABLE of_acme_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    email TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    private_key TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_apply_logs
CREATE TABLE of_apply_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL,
    version TEXT NOT NULL,
    result TEXT NOT NULL,
    message TEXT,
    checksum TEXT NOT NULL DEFAULT '',
    main_config_checksum TEXT NOT NULL DEFAULT '',
    route_config_checksum TEXT NOT NULL DEFAULT '',
    support_file_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL
);
-- table of_cf_connections
CREATE TABLE of_cf_connections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    source TEXT NOT NULL DEFAULT '',
    dns_account_id INTEGER,
    authorization TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    verified_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_cf_pointing_groups
CREATE TABLE of_cf_pointing_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    primary_node_id INTEGER NOT NULL,
    backup_node_id INTEGER,
    active_node_id INTEGER NOT NULL,
    default_proxied BOOLEAN NOT NULL DEFAULT FALSE,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_cf_pointing_members
CREATE TABLE of_cf_pointing_members (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    zone_domain_id INTEGER NOT NULL,
    proxied BOOLEAN NOT NULL DEFAULT FALSE,
    cf_zone_id TEXT NOT NULL DEFAULT '',
    cf_record_id TEXT NOT NULL DEFAULT '',
    desired_ip TEXT NOT NULL DEFAULT '',
    sync_status TEXT NOT NULL DEFAULT 'pending',
    last_error TEXT NOT NULL DEFAULT '',
    synced_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_config_versions
CREATE TABLE "of_config_versions" (
    version VARCHAR(32) PRIMARY KEY,
    snapshot_json TEXT NOT NULL,
    main_config TEXT NOT NULL DEFAULT '',
    rendered_config TEXT NOT NULL,
    support_files_json TEXT NOT NULL DEFAULT '[]',
    checksum VARCHAR(64) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    created_by VARCHAR(64) NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_dns_accounts
CREATE TABLE of_dns_accounts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    authorization TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_access_logs
CREATE TABLE of_node_access_logs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id         TEXT NOT NULL DEFAULT '',
    logged_at       DATETIME NOT NULL,
    remote_addr     TEXT NOT NULL DEFAULT '',
    region          TEXT NOT NULL DEFAULT '',
    host            TEXT NOT NULL DEFAULT '',
    path            TEXT NOT NULL DEFAULT '',
    user_agent      TEXT NOT NULL DEFAULT '',
    cache_status    TEXT NOT NULL DEFAULT '',
    status_code     INTEGER NOT NULL DEFAULT 0,
    bytes_sent      INTEGER NOT NULL DEFAULT 0,
    request_length  INTEGER NOT NULL DEFAULT 0,
    request_time_ms INTEGER NOT NULL DEFAULT 0,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_edge_health
CREATE TABLE of_node_edge_health (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id     TEXT NOT NULL DEFAULT '',
    captured_at DATETIME NOT NULL,
    status      TEXT NOT NULL DEFAULT '',
    connections INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_health_events
CREATE TABLE of_node_health_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    status TEXT NOT NULL,
    message TEXT,
    first_triggered_at DATETIME NOT NULL,
    last_triggered_at DATETIME NOT NULL,
    reported_at DATETIME NOT NULL,
    resolved_at DATETIME,
    metadata_json TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_metric_snapshots
CREATE TABLE of_node_metric_snapshots (
    id                  INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id             TEXT NOT NULL DEFAULT '',
    captured_at         DATETIME NOT NULL,
    cpu_usage_percent   REAL NOT NULL DEFAULT 0,
    memory_used_bytes   INTEGER NOT NULL DEFAULT 0,
    memory_total_bytes  INTEGER NOT NULL DEFAULT 0,
    storage_used_bytes  INTEGER NOT NULL DEFAULT 0,
    storage_total_bytes INTEGER NOT NULL DEFAULT 0,
    disk_read_bytes     INTEGER NOT NULL DEFAULT 0,
    disk_write_bytes    INTEGER NOT NULL DEFAULT 0,
    network_rx_bytes    INTEGER NOT NULL DEFAULT 0,
    network_tx_bytes    INTEGER NOT NULL DEFAULT 0,
    created_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_obs_frpc
CREATE TABLE of_node_obs_frpc (
    id                     INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id                TEXT NOT NULL DEFAULT '',
    captured_at            DATETIME NOT NULL,
    tunnel_status          TEXT NOT NULL DEFAULT '',
    connected_relays_count INTEGER NOT NULL DEFAULT 0,
    created_at             DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_obs_frps
CREATE TABLE of_node_obs_frps (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id           TEXT NOT NULL DEFAULT '',
    captured_at       DATETIME NOT NULL,
    frps_connections  INTEGER NOT NULL DEFAULT 0,
    frps_proxy_count  INTEGER NOT NULL DEFAULT 0,
    frps_client_count INTEGER NOT NULL DEFAULT 0,
    frps_proxies      TEXT NOT NULL DEFAULT '',
    created_at        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_node_system_profiles
CREATE TABLE of_node_system_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL,
    hostname TEXT NOT NULL DEFAULT '',
    os_name TEXT NOT NULL DEFAULT '',
    os_version TEXT NOT NULL DEFAULT '',
    kernel_version TEXT NOT NULL DEFAULT '',
    architecture TEXT NOT NULL DEFAULT '',
    cpu_model TEXT NOT NULL DEFAULT '',
    cpu_cores INTEGER NOT NULL DEFAULT 0,
    total_memory_bytes INTEGER NOT NULL DEFAULT 0,
    total_disk_bytes INTEGER NOT NULL DEFAULT 0,
    uptime_seconds INTEGER NOT NULL DEFAULT 0,
    reported_at DATETIME NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_nodes
CREATE TABLE of_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id TEXT NOT NULL,
    name TEXT NOT NULL,
    ip TEXT NOT NULL DEFAULT '',
    ip_manual_override INTEGER NOT NULL DEFAULT 0,
    geo_name TEXT NOT NULL DEFAULT '',
    geo_latitude REAL,
    geo_longitude REAL,
    geo_manual_override INTEGER NOT NULL DEFAULT 0,
    access_token TEXT NOT NULL DEFAULT '',
    auto_update_enabled INTEGER NOT NULL DEFAULT 0,
    update_requested INTEGER NOT NULL DEFAULT 0,
    update_channel TEXT NOT NULL DEFAULT 'stable',
    update_tag TEXT NOT NULL DEFAULT '',
    restart_openresty_requested INTEGER NOT NULL DEFAULT 0,
    version TEXT NOT NULL DEFAULT '',
    ext_version TEXT NOT NULL DEFAULT '',
    openresty_status TEXT NOT NULL DEFAULT 'unknown',
    openresty_message TEXT,
    status TEXT NOT NULL DEFAULT 'offline',
    current_version TEXT NOT NULL DEFAULT '',
    last_seen_at DATETIME,
    last_error TEXT,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    node_type TEXT NOT NULL DEFAULT 'edge_node',
    relay_bind_port INTEGER NOT NULL DEFAULT 0,
    relay_vhost_http_port INTEGER NOT NULL DEFAULT 0,
    relay_auth_token TEXT NOT NULL DEFAULT '',
    relay_agent_access_addr TEXT NOT NULL DEFAULT '',
    relay_client_access_addr TEXT NOT NULL DEFAULT '',
    relay_client_proxy_url TEXT NOT NULL DEFAULT '',
    capabilities_json TEXT NOT NULL DEFAULT '[]',
    relay_status TEXT NOT NULL DEFAULT 'unknown',
    relay_web_server_enabled INTEGER NOT NULL DEFAULT 0
);
-- table of_origins
CREATE TABLE of_origins (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    address TEXT NOT NULL,
    remark TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_pages_deployment_files
CREATE TABLE of_pages_deployment_files (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    deployment_id INTEGER NOT NULL,
    path TEXT NOT NULL,
    size INTEGER NOT NULL DEFAULT 0,
    checksum TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_pages_deployments
CREATE TABLE of_pages_deployments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    deployment_number INTEGER NOT NULL,
    checksum TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'uploaded',
    artifact_path TEXT NOT NULL,
    file_count INTEGER NOT NULL DEFAULT 0,
    total_size INTEGER NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    activated_at DATETIME
, upload_id INTEGER NOT NULL DEFAULT 0, source_type TEXT NOT NULL DEFAULT '', source_identity TEXT, source_revision TEXT, source_label TEXT NOT NULL DEFAULT '', source_meta TEXT NOT NULL DEFAULT '', trigger_type TEXT NOT NULL DEFAULT '');
-- table of_pages_project_source_runtime
CREATE TABLE of_pages_project_source_runtime (
    source_id INTEGER PRIMARY KEY,
    etag TEXT NOT NULL DEFAULT '',
    last_seen_revision TEXT NOT NULL DEFAULT '',
    last_seen_detail TEXT NOT NULL DEFAULT '',
    last_applied_revision TEXT NOT NULL DEFAULT '',
    last_applied_detail TEXT NOT NULL DEFAULT '',
    sync_status TEXT NOT NULL DEFAULT '',
    last_error TEXT NOT NULL DEFAULT '',
    last_checked_at DATETIME,
    last_synced_at DATETIME,
    next_check_at DATETIME,
    lease_expires_at DATETIME,
    lease_token TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_pages_project_sources
CREATE TABLE of_pages_project_sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL,
    source_type TEXT NOT NULL DEFAULT '',
    remote_url TEXT NOT NULL DEFAULT '',
    allow_insecure INTEGER NOT NULL DEFAULT 0,
    github_repository TEXT NOT NULL DEFAULT '',
    release_selector TEXT NOT NULL DEFAULT '',
    release_tag TEXT NOT NULL DEFAULT '',
    asset_name TEXT NOT NULL DEFAULT '',
    auto_update_enabled INTEGER NOT NULL DEFAULT 0,
    check_interval_minutes INTEGER NOT NULL DEFAULT 0,
    config_version INTEGER NOT NULL DEFAULT 0,
    source_identity TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_pages_projects
CREATE TABLE of_pages_projects (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled INTEGER NOT NULL DEFAULT 1,
    spa_fallback_enabled INTEGER NOT NULL DEFAULT 0,
    spa_fallback_path TEXT NOT NULL DEFAULT '/index.html',
    api_proxy_enabled INTEGER NOT NULL DEFAULT 0,
    api_proxy_path TEXT NOT NULL DEFAULT '',
    api_proxy_pass TEXT NOT NULL DEFAULT '',
    api_proxy_rewrite TEXT NOT NULL DEFAULT '',
    active_deployment_id INTEGER,
    root_dir TEXT NOT NULL DEFAULT '',
    entry_file TEXT NOT NULL DEFAULT 'index.html',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, content_config_version INTEGER NOT NULL DEFAULT 0);
-- table of_proxy_routes
CREATE TABLE "of_proxy_routes" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    site_name TEXT NOT NULL DEFAULT '',
    origin_id INTEGER,
    origin_url TEXT NOT NULL,
    origin_host TEXT NOT NULL DEFAULT '',
    upstreams TEXT NOT NULL DEFAULT '[]',
    enabled INTEGER NOT NULL DEFAULT 1,
    enable_https INTEGER NOT NULL DEFAULT 0,
    redirect_http INTEGER NOT NULL DEFAULT 0,
    limit_conn_per_server INTEGER NOT NULL DEFAULT 0,
    limit_conn_per_ip INTEGER NOT NULL DEFAULT 0,
    limit_rate TEXT NOT NULL DEFAULT '',
    cache_enabled INTEGER NOT NULL DEFAULT 0,
    cache_policy TEXT NOT NULL DEFAULT '',
    cache_rules TEXT NOT NULL DEFAULT '[]',
    custom_headers TEXT NOT NULL DEFAULT '[]',
    basic_auth_enabled INTEGER NOT NULL DEFAULT 0,
    basic_auth_username TEXT NOT NULL DEFAULT '',
    basic_auth_password TEXT NOT NULL DEFAULT '',
    upstream_type TEXT NOT NULL DEFAULT 'direct',
    tunnel_node_id INTEGER,
    tunnel_target_addr TEXT NOT NULL DEFAULT '',
    tunnel_target_protocol TEXT NOT NULL DEFAULT '',
    pages_project_id INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, limit_req_per_ip VARCHAR(32) NOT NULL DEFAULT '');
-- table of_tls_certificates
CREATE TABLE of_tls_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    cert_pem TEXT NOT NULL,
    key_pem TEXT NOT NULL,
    not_before DATETIME,
    not_after DATETIME,
    remark TEXT NOT NULL DEFAULT '',
    provider TEXT NOT NULL DEFAULT 'upload',
    acme_account_id INTEGER NOT NULL DEFAULT 0,
    dns_account_id INTEGER NOT NULL DEFAULT 0,
    key_algorithm TEXT NOT NULL DEFAULT '',
    auto_renew INTEGER NOT NULL DEFAULT 0,
    primary_domain TEXT NOT NULL DEFAULT '',
    other_domains TEXT NOT NULL DEFAULT '',
    disable_cname INTEGER NOT NULL DEFAULT 0,
    skip_dns INTEGER NOT NULL DEFAULT 0,
    dns1 TEXT NOT NULL DEFAULT '',
    dns2 TEXT NOT NULL DEFAULT '',
    apply_status TEXT NOT NULL DEFAULT 'ready',
    apply_message TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_waf_ip_groups
CREATE TABLE of_waf_ip_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    ip_list TEXT NOT NULL DEFAULT '[]',
    auto_config TEXT NOT NULL DEFAULT '{}',
    ext_ips TEXT NOT NULL DEFAULT '[]',
    subscription_url TEXT NOT NULL DEFAULT '',
    subscription_format TEXT NOT NULL DEFAULT 'text',
    subscription_mapping_rule TEXT NOT NULL DEFAULT '',
    sync_interval_minutes INTEGER NOT NULL DEFAULT 1440,
    last_synced_at DATETIME,
    next_sync_at DATETIME,
    last_sync_status TEXT NOT NULL DEFAULT '',
    last_sync_message TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_waf_rule_group_bindings
CREATE TABLE of_waf_rule_group_bindings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    rule_group_id INTEGER NOT NULL,
    proxy_route_id INTEGER NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, sequence INTEGER NOT NULL DEFAULT 0);
-- table of_waf_rule_groups
CREATE TABLE of_waf_rule_groups (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_global BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
, graph TEXT NOT NULL DEFAULT '', revision INTEGER NOT NULL DEFAULT 1);
-- table of_zone_domains
CREATE TABLE of_zone_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    zone_id INTEGER NOT NULL,
    proxy_route_id INTEGER,
    domain TEXT NOT NULL,
    cert_id INTEGER,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table of_zones
CREATE TABLE of_zones (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table w_access_tokens
CREATE TABLE "w_access_tokens" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id BIGINT NOT NULL,
    name VARCHAR(128) NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    masked_token VARCHAR(64) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
, is_admin BOOLEAN NOT NULL DEFAULT 0);
-- table w_auth_sources
CREATE TABLE "w_auth_sources" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(80) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL,
    display_name VARCHAR(100),
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    client_id VARCHAR(255),
    client_secret VARCHAR(1024),
    openid_discovery_url VARCHAR(1024),
    scopes VARCHAR(255),
    icon_url VARCHAR(1024),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- table w_external_accounts
CREATE TABLE "w_external_accounts" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    auth_source_id BIGINT,
    user_id BIGINT NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    external_username VARCHAR(255),
    email VARCHAR(255),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- table w_push_channels
CREATE TABLE w_push_channels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    type TEXT NOT NULL DEFAULT 'custom',
    token TEXT,
    url TEXT NOT NULL,
    other TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);
-- table w_push_events
CREATE TABLE w_push_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_key TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    channels TEXT NOT NULL,
    targets TEXT NOT NULL,
    template TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
, task_type VARCHAR(100) NOT NULL DEFAULT '');
-- table w_push_histories
CREATE TABLE w_push_histories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_key TEXT NOT NULL,
    channel TEXT NOT NULL,
    target TEXT NOT NULL,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    level TEXT NOT NULL,
    status TEXT NOT NULL,
    error_msg TEXT,
    created_at DATETIME NOT NULL
);
-- table w_schedules
CREATE TABLE "w_schedules" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name VARCHAR(128) NOT NULL,
    task_type VARCHAR(64) NOT NULL,
    cron VARCHAR(64) NOT NULL,
    payload TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- table w_schedules_backup_of_database_auto_cleanup
CREATE TABLE w_schedules_backup_of_database_auto_cleanup(
  id INT,
  name TEXT,
  task_type TEXT,
  cron TEXT,
  payload TEXT,
  is_active NUM,
  created_at NUM,
  updated_at NUM
);
-- table w_system_configs
CREATE TABLE "w_system_configs" (
    key VARCHAR(64) PRIMARY KEY,
    value TEXT NOT NULL,
    type VARCHAR(32) NOT NULL DEFAULT 'system',
    visibility INTEGER NOT NULL DEFAULT 0,
    description VARCHAR(255),
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- table w_task_executions
CREATE TABLE "w_task_executions" (
    id BIGINT PRIMARY KEY,
    task_id VARCHAR(128) NOT NULL UNIQUE,
    task_type VARCHAR(64) NOT NULL,
    task_name VARCHAR(128),
    status VARCHAR(32) NOT NULL,
    retryable BOOLEAN NOT NULL DEFAULT FALSE,
    max_retry INTEGER NOT NULL DEFAULT 0,
    retry_count INTEGER NOT NULL DEFAULT 0,
    log TEXT,
    error_message TEXT,
    result TEXT,
    started_at DATETIME,
    finished_at DATETIME,
    duration BIGINT,
    payload TEXT,
    triggered_by VARCHAR(32) NOT NULL DEFAULT 'system',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- table w_templates
CREATE TABLE "w_templates" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key VARCHAR(80) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL DEFAULT 'email',
    subject VARCHAR(255),
    content TEXT NOT NULL,
    description VARCHAR(255),
    is_system BOOLEAN NOT NULL DEFAULT FALSE,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- table w_upload_stats
CREATE TABLE w_upload_stats (
    dimension VARCHAR(32) NOT NULL,
    stat_key VARCHAR(64) NOT NULL DEFAULT '',
    file_count BIGINT NOT NULL DEFAULT 0,
    file_size BIGINT NOT NULL DEFAULT 0,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (dimension, stat_key)
);
-- table w_uploads
CREATE TABLE "w_uploads" (
    id BIGINT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    file_name VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL,
    file_size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    extension VARCHAR(50) NOT NULL,
    hash VARCHAR(64),
    type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL,
    metadata JSON,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
, access_mode INTEGER NOT NULL DEFAULT 0);
-- table w_user_access_logs
CREATE TABLE w_user_access_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER NOT NULL DEFAULT 0,
    path       TEXT NOT NULL DEFAULT '',
    method     TEXT NOT NULL DEFAULT '',
    ip         TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    headers    TEXT NOT NULL DEFAULT '',
    status     INTEGER NOT NULL DEFAULT 0,
    latency    INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- table w_users
CREATE TABLE "w_users" (
    id BIGINT PRIMARY KEY,
    username VARCHAR(64) UNIQUE,
    password VARCHAR(255),
    nickname VARCHAR(255),
    email VARCHAR(255),
    avatar_url VARCHAR(255),
    is_active BOOLEAN DEFAULT TRUE,
    is_admin BOOLEAN DEFAULT FALSE,
    bio VARCHAR(500),
    phone VARCHAR(32),
    gender VARCHAR(16),
    website VARCHAR(255),
    location VARCHAR(255),
    last_login_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
-- total objects (excl. bookkeeping): 130
