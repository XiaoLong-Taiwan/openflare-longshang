-- +goose Up
ALTER TABLE of_proxy_routes
    ADD COLUMN upstream_targets TEXT NOT NULL DEFAULT '[]',
    ADD COLUMN load_balancing VARCHAR(32) NOT NULL DEFAULT 'round_robin';

-- +goose Down
ALTER TABLE of_proxy_routes
    DROP COLUMN load_balancing,
    DROP COLUMN upstream_targets;
