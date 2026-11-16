-- +goose Up
DELETE FROM w_schedules
WHERE task_type IN ('cloudflare:sync_all_groups', 'of_cloudflare_sync_all_groups')
  AND id NOT IN (
    SELECT id FROM w_schedules
    WHERE task_type IN ('cloudflare:sync_all_groups', 'of_cloudflare_sync_all_groups')
    ORDER BY id
    LIMIT 1
  );
UPDATE w_schedules
SET task_type = 'cloudflare:sync_all_groups'
WHERE task_type = 'of_cloudflare_sync_all_groups';
INSERT INTO w_schedules (name, task_type, cron, payload, is_active, created_at, updated_at)
SELECT 'OpenFlare Cloudflare 分组健康同步', 'cloudflare:sync_all_groups', '0 */4 * * *', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM w_schedules WHERE task_type = 'cloudflare:sync_all_groups');

-- +goose Down
SELECT 1;
