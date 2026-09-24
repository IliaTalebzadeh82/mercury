#!/bin/sh
set -eu

migration_database=mercury_migration_test

case "$MERCURY_MIGRATION_TEST_DATABASE_URL" in
    */"$migration_database"|*/"$migration_database"\?*) ;;
    *)
        echo "MERCURY_MIGRATION_TEST_DATABASE_URL must target $migration_database" >&2
        exit 1
        ;;
esac

reset_database() {
    docker compose exec -T postgres dropdb --if-exists --force -U mercury "$migration_database"
    docker compose exec -T postgres createdb -U mercury "$migration_database"
}

goose() {
    go run -tags="$GOOSE_TAGS" "github.com/pressly/goose/v3/cmd/goose@$GOOSE_VERSION" \
        -dir testdata/migrations postgres "$MERCURY_MIGRATION_TEST_DATABASE_URL" "$@"
}

assert_state() {
    expected_version=$1
    expected_table=$2
    actual_version=$(docker compose exec -T postgres psql -U mercury -d "$migration_database" -tAc \
        "SELECT COALESCE(MAX(version_id) FILTER (WHERE is_applied), 0) FROM goose_db_version")
    actual_table=$(docker compose exec -T postgres psql -U mercury -d "$migration_database" -tAc \
        "SELECT to_regclass('public.phase_zero_migration_verification') IS NOT NULL")

    if [ "$actual_version" != "$expected_version" ] || [ "$actual_table" != "$expected_table" ]; then
        echo "migration verification state mismatch" >&2
        exit 1
    fi
}

trap reset_database EXIT
reset_database

goose up
assert_state 1 t

goose down
assert_state 0 f

goose up
assert_state 1 t
