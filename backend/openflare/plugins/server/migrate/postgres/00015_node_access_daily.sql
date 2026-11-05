-- +goose Up
CREATE TABLE IF NOT EXISTS of_node_access_daily (
    day VARCHAR(10) NOT NULL, node_id TEXT NOT NULL, status_code INTEGER NOT NULL DEFAULT 0,
    source TEXT NOT NULL, request_count BIGINT NOT NULL DEFAULT 0,
    bytes_sent BIGINT NOT NULL DEFAULT 0, request_length BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, node_id, status_code, source)
);
CREATE TABLE IF NOT EXISTS of_node_access_quota (
    day VARCHAR(10) NOT NULL, node_id TEXT NOT NULL,
    used_bytes BIGINT NOT NULL DEFAULT 0, seen_bytes BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, node_id)
);
CREATE TABLE IF NOT EXISTS of_node_access_batches (batch_key TEXT PRIMARY KEY);
ALTER TABLE of_node_access_logs ADD COLUMN IF NOT EXISTS summary_tracked BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE of_node_access_logs DROP COLUMN IF EXISTS summary_tracked;
DROP TABLE IF EXISTS of_node_access_batches;
DROP TABLE IF EXISTS of_node_access_quota;
DROP TABLE IF EXISTS of_node_access_daily;
