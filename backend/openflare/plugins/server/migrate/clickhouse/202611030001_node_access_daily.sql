-- +goose Up
ALTER TABLE of_node_access_logs ADD COLUMN IF NOT EXISTS summary_tracked Bool DEFAULT false;

-- +goose Down
ALTER TABLE of_node_access_logs DROP COLUMN IF EXISTS summary_tracked;
