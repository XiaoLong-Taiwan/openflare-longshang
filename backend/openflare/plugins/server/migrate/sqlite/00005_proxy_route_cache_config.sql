-- +goose Up
ALTER TABLE of_proxy_routes ADD COLUMN cache_config TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE of_proxy_routes DROP COLUMN cache_config;
