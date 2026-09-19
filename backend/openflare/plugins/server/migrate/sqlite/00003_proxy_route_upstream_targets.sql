-- +goose Up
ALTER TABLE of_proxy_routes ADD COLUMN upstream_targets TEXT NOT NULL DEFAULT '[]';
ALTER TABLE of_proxy_routes ADD COLUMN load_balancing TEXT NOT NULL DEFAULT 'round_robin';

-- +goose Down
ALTER TABLE of_proxy_routes DROP COLUMN load_balancing;
ALTER TABLE of_proxy_routes DROP COLUMN upstream_targets;
