-- +goose Up
ALTER TABLE of_node_access_batches ADD COLUMN created_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_node_access_batches_created ON of_node_access_batches (created_at);
CREATE INDEX IF NOT EXISTS idx_node_access_pending ON of_node_access_logs (summary_tracked, logged_at, id);
CREATE TABLE IF NOT EXISTS of_node_access_checkpoints (name TEXT PRIMARY KEY);
ALTER TABLE of_node_access_quota ADD COLUMN sample_boundary TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE of_node_access_quota DROP COLUMN sample_boundary;
DROP TABLE IF EXISTS of_node_access_checkpoints;
DROP INDEX IF EXISTS idx_node_access_pending;
DROP INDEX IF EXISTS idx_node_access_batches_created;
ALTER TABLE of_node_access_batches DROP COLUMN created_at;
