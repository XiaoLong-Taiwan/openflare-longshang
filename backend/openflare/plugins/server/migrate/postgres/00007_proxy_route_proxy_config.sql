-- +goose Up
ALTER TABLE of_proxy_routes
    ADD COLUMN proxy_config TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE of_proxy_routes
    DROP COLUMN proxy_config;
