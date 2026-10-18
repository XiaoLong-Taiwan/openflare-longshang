-- +goose Up
INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES ('trusted_proxy_cidrs', '{"enabled":true,"mode":"append","cidrs":[]}', 'system', 0, '可信代理范围（JSON，默认包含 Cloudflare 官方网段）', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM w_system_configs WHERE key = 'trusted_proxy_cidrs';
