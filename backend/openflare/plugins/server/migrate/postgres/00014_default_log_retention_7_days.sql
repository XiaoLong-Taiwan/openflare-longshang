-- +goose Up
UPDATE w_system_configs
SET value = '7', updated_at = CURRENT_TIMESTAMP
WHERE key IN (
    'log_retention_days_postgres',
    'log_retention_days_sqlite',
    'log_retention_days_clickhouse'
) AND value = '30';

-- +goose Down
SELECT 1;
