-- +goose Up
ALTER TABLE of_cf_pointing_groups ADD COLUMN debounce_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE of_cf_pointing_group_nodes ADD COLUMN ipv6 VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE of_cf_pointing_managed_records ADD COLUMN record_type VARCHAR(16) NOT NULL DEFAULT 'A';

-- +goose Down
ALTER TABLE of_cf_pointing_managed_records DROP COLUMN record_type;
ALTER TABLE of_cf_pointing_group_nodes DROP COLUMN ipv6;
ALTER TABLE of_cf_pointing_groups DROP COLUMN debounce_seconds;
