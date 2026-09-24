-- +goose Up
ALTER TABLE of_zone_domains
    ADD COLUMN nginx_config TEXT NOT NULL DEFAULT '{}';

-- +goose Down
ALTER TABLE of_zone_domains
    DROP COLUMN nginx_config;
