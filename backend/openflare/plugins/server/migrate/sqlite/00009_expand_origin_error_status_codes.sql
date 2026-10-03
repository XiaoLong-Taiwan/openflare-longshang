-- +goose Up
UPDATE w_system_configs
SET value = '["400-599"]', updated_at = CURRENT_TIMESTAMP
WHERE key = 'origin_error_page_status_codes'
  AND value = '["500-599"]';

-- +goose Down
SELECT 1;
