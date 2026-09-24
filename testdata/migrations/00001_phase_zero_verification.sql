-- +goose Up
CREATE TABLE phase_zero_migration_verification (
    id integer PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE phase_zero_migration_verification;
