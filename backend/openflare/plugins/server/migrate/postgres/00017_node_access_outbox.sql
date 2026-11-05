-- +goose Up
ALTER TABLE of_node_access_logs ADD COLUMN IF NOT EXISTS sample_key TEXT NOT NULL DEFAULT '';
ALTER TABLE of_node_access_logs ADD COLUMN IF NOT EXISTS detail_size BIGINT NOT NULL DEFAULT 0;
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_class WHERE oid = 'of_node_access_logs'::regclass AND relkind = 'p') THEN
        CREATE INDEX IF NOT EXISTS idx_node_access_sample ON ONLY of_node_access_logs (node_id, logged_at, sample_key);
    END IF;
END
$$;
-- +goose StatementEnd
CREATE TABLE IF NOT EXISTS of_node_access_outbox (id BIGINT PRIMARY KEY, payload TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP);
ALTER TABLE of_node_access_checkpoints ADD COLUMN IF NOT EXISTS through_id BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE of_node_access_checkpoints DROP COLUMN IF EXISTS through_id;
DROP TABLE IF EXISTS of_node_access_outbox;
DROP INDEX IF EXISTS idx_node_access_sample;
ALTER TABLE of_node_access_logs DROP COLUMN IF EXISTS sample_key;
ALTER TABLE of_node_access_logs DROP COLUMN IF EXISTS detail_size;
