-- +goose Up
ALTER TABLE of_cf_pointing_groups ADD COLUMN target_mode VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE of_cf_pointing_groups ADD COLUMN record_type VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE of_cf_pointing_groups ADD COLUMN record_content VARCHAR(253) NOT NULL DEFAULT '';
ALTER TABLE of_cf_pointing_members ALTER COLUMN desired_ip TYPE VARCHAR(253);
ALTER TABLE of_cf_pointing_managed_records ALTER COLUMN desired_ip TYPE VARCHAR(253);

-- +goose Down
ALTER TABLE of_cf_pointing_groups DROP COLUMN record_content;
ALTER TABLE of_cf_pointing_groups DROP COLUMN record_type;
ALTER TABLE of_cf_pointing_groups DROP COLUMN target_mode;
