-- +goose Up
INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES
    ('log_retention_days_clickhouse', '7', 'business', 0, 'ClickHouse 原始日志保留天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('log_retention_days_postgres', '7', 'business', 0, 'PostgreSQL 原始日志保留天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('log_retention_days_sqlite', '7', 'business', 0, 'SQLite 原始日志保留天数', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

-- +goose Down
SELECT 1;
