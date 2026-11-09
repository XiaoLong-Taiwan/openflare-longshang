-- +goose Up
ALTER TABLE of_cf_pointing_groups ADD COLUMN target_mode VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE of_cf_pointing_groups ADD COLUMN record_type VARCHAR(16) NOT NULL DEFAULT '';
ALTER TABLE of_cf_pointing_groups ADD COLUMN record_content VARCHAR(253) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE of_cf_pointing_groups DROP COLUMN record_content;
ALTER TABLE of_cf_pointing_groups DROP COLUMN record_type;
ALTER TABLE of_cf_pointing_groups DROP COLUMN target_mode;
