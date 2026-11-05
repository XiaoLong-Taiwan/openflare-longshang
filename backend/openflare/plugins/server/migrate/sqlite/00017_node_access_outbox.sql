-- +goose Up
ALTER TABLE of_node_access_logs ADD COLUMN sample_key TEXT NOT NULL DEFAULT '';
ALTER TABLE of_node_access_logs ADD COLUMN detail_size BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_node_access_sample ON of_node_access_logs (node_id, logged_at, sample_key);
CREATE TABLE IF NOT EXISTS of_node_access_outbox (id BIGINT PRIMARY KEY, payload TEXT NOT NULL DEFAULT '', created_at DATETIME);
ALTER TABLE of_node_access_checkpoints ADD COLUMN through_id BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE of_node_access_checkpoints DROP COLUMN through_id;
DROP TABLE IF EXISTS of_node_access_outbox;
DROP INDEX IF EXISTS idx_node_access_sample;
ALTER TABLE of_node_access_logs DROP COLUMN sample_key;
ALTER TABLE of_node_access_logs DROP COLUMN detail_size;
