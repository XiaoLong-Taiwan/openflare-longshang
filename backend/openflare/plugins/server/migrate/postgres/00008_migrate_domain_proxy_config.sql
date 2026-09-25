-- +goose Up
WITH candidates AS (
    SELECT
        zd.proxy_route_id AS route_id,
        zd.nginx_config::jsonb AS config
    FROM of_zone_domains zd
    WHERE zd.proxy_route_id IS NOT NULL
      AND zd.nginx_config IS NOT NULL
      AND zd.nginx_config <> '{}'
      AND (
          SELECT COUNT(*)
          FROM of_zone_domains same_route
          WHERE same_route.proxy_route_id = zd.proxy_route_id
      ) = 1
)
UPDATE of_proxy_routes route
SET proxy_config = jsonb_strip_nulls(jsonb_build_object(
    'proxy_connect_timeout', CASE
        WHEN candidate.config->>'proxy_connect_timeout' ~ '^[0-9]+$'
             AND (candidate.config->>'proxy_connect_timeout')::integer > 0
        THEN to_jsonb((candidate.config->>'proxy_connect_timeout') || 's')
    END,
    'proxy_send_timeout', CASE
        WHEN candidate.config->>'proxy_send_timeout' ~ '^[0-9]+$'
             AND (candidate.config->>'proxy_send_timeout')::integer > 0
        THEN to_jsonb((candidate.config->>'proxy_send_timeout') || 's')
    END,
    'proxy_read_timeout', CASE
        WHEN candidate.config->>'proxy_read_timeout' ~ '^[0-9]+$'
             AND (candidate.config->>'proxy_read_timeout')::integer > 0
        THEN to_jsonb((candidate.config->>'proxy_read_timeout') || 's')
    END,
    'client_header_timeout', CASE
        WHEN candidate.config->>'client_header_timeout' ~ '^[0-9]+$'
             AND (candidate.config->>'client_header_timeout')::integer > 0
        THEN to_jsonb((candidate.config->>'client_header_timeout') || 's')
    END,
    'client_body_timeout', CASE
        WHEN candidate.config->>'client_body_timeout' ~ '^[0-9]+$'
             AND (candidate.config->>'client_body_timeout')::integer > 0
        THEN to_jsonb((candidate.config->>'client_body_timeout') || 's')
    END,
    'send_timeout', CASE
        WHEN candidate.config->>'send_timeout' ~ '^[0-9]+$'
             AND (candidate.config->>'send_timeout')::integer > 0
        THEN to_jsonb((candidate.config->>'send_timeout') || 's')
    END,
    'client_max_body_size', NULLIF(candidate.config->>'client_max_body_size', ''),
    'websocket_enabled', candidate.config->'websocket_enabled',
    'proxy_request_buffering', candidate.config->'proxy_request_buffering',
    'proxy_buffering_enabled', candidate.config->'proxy_buffering_enabled',
    'custom_headers', COALESCE(candidate.config->'custom_headers', '[]'::jsonb)
))::text
FROM candidates candidate
WHERE route.id = candidate.route_id
  AND route.proxy_config = '{}';

-- +goose Down
UPDATE of_proxy_routes
SET proxy_config = '{}'
WHERE proxy_config <> '{}';
