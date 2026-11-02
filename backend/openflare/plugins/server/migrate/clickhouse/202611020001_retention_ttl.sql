-- +goose Up
ALTER TABLE of_node_access_logs MODIFY TTL toDateTime(logged_at) + INTERVAL 90 DAY;
ALTER TABLE of_node_metric_snapshots MODIFY TTL toDateTime(captured_at) + INTERVAL 30 DAY;
ALTER TABLE of_node_edge_health MODIFY TTL toDateTime(captured_at) + INTERVAL 30 DAY;
ALTER TABLE of_node_obs_frps MODIFY TTL toDateTime(captured_at) + INTERVAL 30 DAY;
ALTER TABLE of_node_obs_frpc MODIFY TTL toDateTime(captured_at) + INTERVAL 30 DAY;

-- +goose Down
ALTER TABLE of_node_access_logs MODIFY TTL toDateTime(logged_at) + INTERVAL 7 DAY;
ALTER TABLE of_node_metric_snapshots MODIFY TTL toDateTime(captured_at) + INTERVAL 3 DAY;
ALTER TABLE of_node_edge_health MODIFY TTL toDateTime(captured_at) + INTERVAL 3 DAY;
ALTER TABLE of_node_obs_frps MODIFY TTL toDateTime(captured_at) + INTERVAL 3 DAY;
ALTER TABLE of_node_obs_frpc MODIFY TTL toDateTime(captured_at) + INTERVAL 3 DAY;
