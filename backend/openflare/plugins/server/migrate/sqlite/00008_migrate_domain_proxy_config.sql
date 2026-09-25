-- +goose Up
WITH candidates AS (
    SELECT
        zd.proxy_route_id AS route_id,
        zd.nginx_config AS config
    FROM of_zone_domains zd
    WHERE zd.proxy_route_id IS NOT NULL
      AND zd.nginx_config IS NOT NULL
      AND zd.nginx_config <> '{}'
      AND json_valid(zd.nginx_config)
      AND (
          SELECT COUNT(*)
          FROM of_zone_domains same_route
          WHERE same_route.proxy_route_id = zd.proxy_route_id
      ) = 1
)
UPDATE of_proxy_routes
SET proxy_config = (
    SELECT json_patch('{}', json_object(
        'proxy_connect_timeout', CASE
            WHEN json_type(candidate.config, '$.proxy_connect_timeout') = 'integer'
                AND json_extract(candidate.config, '$.proxy_connect_timeout') > 0
            THEN printf('%ds', json_extract(candidate.config, '$.proxy_connect_timeout'))
        END,
        'proxy_send_timeout', CASE
            WHEN json_type(candidate.config, '$.proxy_send_timeout') = 'integer'
                AND json_extract(candidate.config, '$.proxy_send_timeout') > 0
            THEN printf('%ds', json_extract(candidate.config, '$.proxy_send_timeout'))
        END,
        'proxy_read_timeout', CASE
            WHEN json_type(candidate.config, '$.proxy_read_timeout') = 'integer'
                AND json_extract(candidate.config, '$.proxy_read_timeout') > 0
            THEN printf('%ds', json_extract(candidate.config, '$.proxy_read_timeout'))
        END,
        'client_header_timeout', CASE
            WHEN json_type(candidate.config, '$.client_header_timeout') = 'integer'
                AND json_extract(candidate.config, '$.client_header_timeout') > 0
            THEN printf('%ds', json_extract(candidate.config, '$.client_header_timeout'))
        END,
        'client_body_timeout', CASE
            WHEN json_type(candidate.config, '$.client_body_timeout') = 'integer'
                AND json_extract(candidate.config, '$.client_body_timeout') > 0
            THEN printf('%ds', json_extract(candidate.config, '$.client_body_timeout'))
        END,
        'send_timeout', CASE
            WHEN json_type(candidate.config, '$.send_timeout') = 'integer'
                AND json_extract(candidate.config, '$.send_timeout') > 0
            THEN printf('%ds', json_extract(candidate.config, '$.send_timeout'))
        END,
        'client_max_body_size', NULLIF(json_extract(candidate.config, '$.client_max_body_size'), ''),
        'websocket_enabled', json_extract(candidate.config, '$.websocket_enabled'),
        'proxy_request_buffering', json_extract(candidate.config, '$.proxy_request_buffering'),
        'proxy_buffering_enabled', json_extract(candidate.config, '$.proxy_buffering_enabled'),
        'custom_headers', json(COALESCE(json_extract(candidate.config, '$.custom_headers'), '[]'))
    ))
    FROM candidates candidate
    WHERE candidate.route_id = of_proxy_routes.id
)
WHERE id IN (SELECT route_id FROM candidates)
  AND proxy_config = '{}';

-- +goose Down
UPDATE of_proxy_routes
SET proxy_config = '{}'
WHERE proxy_config <> '{}';
