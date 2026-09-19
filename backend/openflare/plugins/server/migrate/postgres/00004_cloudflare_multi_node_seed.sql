-- +goose Up
INSERT INTO of_cf_pointing_group_nodes (group_id, node_id, priority)
SELECT id, primary_node_id, 0 FROM of_cf_pointing_groups
ON CONFLICT (group_id, node_id) DO NOTHING;

INSERT INTO of_cf_pointing_group_nodes (group_id, node_id, priority)
SELECT id, backup_node_id, 1 FROM of_cf_pointing_groups WHERE backup_node_id IS NOT NULL
ON CONFLICT (group_id, node_id) DO NOTHING;

INSERT INTO of_cf_pointing_managed_records (member_id, node_id, cf_record_id, desired_ip)
SELECT members.id, groups.active_node_id, members.cf_record_id, members.desired_ip
FROM of_cf_pointing_members members
JOIN of_cf_pointing_groups groups ON groups.id = members.group_id
WHERE members.cf_record_id <> ''
ON CONFLICT (member_id, node_id) DO NOTHING;

-- +goose Down
SELECT 1;
