-- +goose Up
ALTER TABLE of_node_access_outbox ADD COLUMN IF NOT EXISTS attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE of_node_access_outbox ADD COLUMN IF NOT EXISTS next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE of_node_access_outbox ADD COLUMN IF NOT EXISTS last_error TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_node_access_outbox_retry ON of_node_access_outbox (next_attempt_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_node_access_outbox_retry;
ALTER TABLE of_node_access_outbox DROP COLUMN IF EXISTS last_error;
ALTER TABLE of_node_access_outbox DROP COLUMN IF EXISTS next_attempt_at;
ALTER TABLE of_node_access_outbox DROP COLUMN IF EXISTS attempts;
