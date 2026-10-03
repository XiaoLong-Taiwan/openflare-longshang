-- +goose Up
INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES ('origin_health_check_accept_4xx', 'true', 'business', 0, '源站健康检查是否将 4xx 响应视为健康', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM w_system_configs WHERE key = 'origin_health_check_accept_4xx';
