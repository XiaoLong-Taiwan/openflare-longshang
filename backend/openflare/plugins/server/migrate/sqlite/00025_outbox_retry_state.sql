-- +goose Up
ALTER TABLE of_node_access_outbox ADD COLUMN attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE of_node_access_outbox ADD COLUMN next_attempt_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE of_node_access_outbox ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_node_access_outbox_retry ON of_node_access_outbox (next_attempt_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_node_access_outbox_retry;
ALTER TABLE of_node_access_outbox DROP COLUMN last_error;
ALTER TABLE of_node_access_outbox DROP COLUMN next_attempt_at;
ALTER TABLE of_node_access_outbox DROP COLUMN attempts;
