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
        -dir migrations postgres "$MERCURY_MIGRATION_TEST_DATABASE_URL" "$@"
}

psql_migration() {
    docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U mercury -d "$migration_database" "$@"
}

assert_up() {
    tables=$(psql_migration -tAc \
        "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename IN ('advertisers','placements','campaigns','campaign_target_countries')")
    seeds=$(psql_migration -tAc \
        "SELECT string_agg(code, ',' ORDER BY code) FROM placements")
    [ "$tables" = "4" ] || { echo "expected four Phase 1 tables" >&2; exit 1; }
    [ "$seeds" = "home_feed,restaurant_list,search_results" ] || { echo "placement seeds mismatch" >&2; exit 1; }
}

assert_down() {
    tables=$(psql_migration -tAc \
        "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename IN ('advertisers','placements','campaigns','campaign_target_countries')")
    [ "$tables" = "0" ] || { echo "Phase 1 tables remain after migration down" >&2; exit 1; }
}

assert_country_invariant() {
    psql_migration -c "INSERT INTO advertisers (id,name,creation_idempotency_key,creation_request_fingerprint) VALUES ('00000000-0000-4000-8000-000000000001','Migration advertiser','migration-advertiser',decode(repeat('00',32),'hex'))"

    if psql_migration -c "BEGIN; INSERT INTO campaigns (id,advertiser_id,name,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint) VALUES ('00000000-0000-4000-8000-000000000002','00000000-0000-4000-8000-000000000001','Missing target','home_feed',100,'EUR','missing-target',decode(repeat('01',32),'hex')); COMMIT"; then
        echo "targetless campaign commit unexpectedly succeeded" >&2
        exit 1
    fi

    psql_migration -c "BEGIN; INSERT INTO campaigns (id,advertiser_id,name,placement_code,budget_amount_minor,currency,creation_idempotency_key,creation_request_fingerprint) VALUES ('00000000-0000-4000-8000-000000000003','00000000-0000-4000-8000-000000000001','Valid target','home_feed',100,'EUR','valid-target',decode(repeat('02',32),'hex')); INSERT INTO campaign_target_countries (campaign_id,country_code) VALUES ('00000000-0000-4000-8000-000000000003','DE'); COMMIT"

    if psql_migration -c "BEGIN; DELETE FROM campaign_target_countries WHERE campaign_id='00000000-0000-4000-8000-000000000003'; COMMIT"; then
        echo "deleting the final country target unexpectedly succeeded" >&2
        exit 1
    fi
}

trap reset_database EXIT
reset_database

goose up
assert_up
assert_country_invariant

goose down-to 0
assert_down

goose up
assert_up
