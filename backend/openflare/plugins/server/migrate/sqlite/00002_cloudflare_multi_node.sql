-- +goose Up
CREATE TABLE IF NOT EXISTS of_cf_pointing_group_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    group_id INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_of_cf_pointing_group_nodes_group_node ON of_cf_pointing_group_nodes (group_id, node_id);
CREATE INDEX IF NOT EXISTS idx_of_cf_pointing_group_nodes_group_priority ON of_cf_pointing_group_nodes (group_id, priority);
CREATE INDEX IF NOT EXISTS idx_of_cf_pointing_group_nodes_node_id ON of_cf_pointing_group_nodes (node_id);

CREATE TABLE IF NOT EXISTS of_cf_pointing_managed_records (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    member_id INTEGER NOT NULL,
    node_id INTEGER NOT NULL,
    cf_record_id VARCHAR(64) NOT NULL DEFAULT '',
    desired_ip VARCHAR(64) NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_of_cf_pointing_managed_records_member_node ON of_cf_pointing_managed_records (member_id, node_id);
CREATE INDEX IF NOT EXISTS idx_of_cf_pointing_managed_records_member_id ON of_cf_pointing_managed_records (member_id);

-- +goose Down
DROP TABLE IF EXISTS of_cf_pointing_managed_records;
DROP TABLE IF EXISTS of_cf_pointing_group_nodes;
