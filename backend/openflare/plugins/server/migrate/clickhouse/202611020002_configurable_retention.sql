-- +goose Up
ALTER TABLE of_node_access_logs REMOVE TTL;
ALTER TABLE of_node_metric_snapshots REMOVE TTL;
ALTER TABLE of_node_edge_health REMOVE TTL;
ALTER TABLE of_node_obs_frps REMOVE TTL;
ALTER TABLE of_node_obs_frpc REMOVE TTL;

-- +goose Down
SELECT 1;
