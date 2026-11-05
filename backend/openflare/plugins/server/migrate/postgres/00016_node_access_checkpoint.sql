-- +goose Up
ALTER TABLE of_node_access_batches ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP;
CREATE INDEX IF NOT EXISTS idx_node_access_batches_created ON of_node_access_batches (created_at);
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_class WHERE oid = 'of_node_access_logs'::regclass AND relkind = 'p') THEN
        CREATE INDEX IF NOT EXISTS idx_node_access_pending ON ONLY of_node_access_logs (summary_tracked, logged_at, id);
    END IF;
END
$$;
-- +goose StatementEnd
CREATE TABLE IF NOT EXISTS of_node_access_checkpoints (name TEXT PRIMARY KEY);
ALTER TABLE of_node_access_quota ADD COLUMN IF NOT EXISTS sample_boundary TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE of_node_access_quota DROP COLUMN IF EXISTS sample_boundary;
DROP TABLE IF EXISTS of_node_access_checkpoints;
DROP INDEX IF EXISTS idx_node_access_pending;
DROP INDEX IF EXISTS idx_node_access_batches_created;
ALTER TABLE of_node_access_batches DROP COLUMN IF EXISTS created_at;
