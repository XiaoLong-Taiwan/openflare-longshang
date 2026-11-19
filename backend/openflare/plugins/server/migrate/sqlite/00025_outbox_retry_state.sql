-- +goose Up
CREATE TABLE of_node_access_outbox_retry (
    id BIGINT PRIMARY KEY,
    payload TEXT NOT NULL DEFAULT '',
    created_at DATETIME,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT NOT NULL DEFAULT ''
);
INSERT INTO of_node_access_outbox_retry (id, payload, created_at)
SELECT id, payload, created_at FROM of_node_access_outbox;
DROP TABLE of_node_access_outbox;
ALTER TABLE of_node_access_outbox_retry RENAME TO of_node_access_outbox;
CREATE INDEX IF NOT EXISTS idx_node_access_outbox_retry ON of_node_access_outbox (next_attempt_at, id);

-- +goose Down
DROP INDEX IF EXISTS idx_node_access_outbox_retry;
ALTER TABLE of_node_access_outbox DROP COLUMN last_error;
ALTER TABLE of_node_access_outbox DROP COLUMN next_attempt_at;
ALTER TABLE of_node_access_outbox DROP COLUMN attempts;
