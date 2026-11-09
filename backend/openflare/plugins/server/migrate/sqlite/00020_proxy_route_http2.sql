-- +goose Up
ALTER TABLE of_proxy_routes ADD COLUMN enable_http2 INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE of_proxy_routes DROP COLUMN enable_http2;
