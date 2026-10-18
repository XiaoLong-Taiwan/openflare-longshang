-- +goose Up
CREATE TABLE IF NOT EXISTS w_user_access_logs (
    id         BIGINT NOT NULL,
    user_id    BIGINT NOT NULL DEFAULT 0,
    path       VARCHAR(2048) NOT NULL DEFAULT '',
    method     VARCHAR(16) NOT NULL DEFAULT '',
    ip         VARCHAR(128) NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    headers    TEXT NOT NULL DEFAULT '',
    status     INTEGER NOT NULL DEFAULT 0,
    latency    BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

CREATE INDEX IF NOT EXISTS idx_w_user_access_logs_user_created
    ON w_user_access_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_w_user_access_logs_created
    ON w_user_access_logs (created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_w_user_access_logs_ip
    ON w_user_access_logs (ip, created_at DESC);

-- +goose StatementBegin
DO $$
DECLARE
    d date;
BEGIN
    FOR d IN SELECT generate_series(date_trunc('month', now())::date, (date_trunc('month', now()) + interval '2 months')::date, interval '1 month')::date
    LOOP
        EXECUTE format('CREATE TABLE IF NOT EXISTS w_user_access_logs_%s PARTITION OF w_user_access_logs FOR VALUES FROM (%L) TO (%L)',
            to_char(d, 'YYYYMM'), d, d + interval '1 month');
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TABLE IF EXISTS w_user_access_logs;
