#!/bin/sh
set -eu

performance_database=mercury_performance_test

case "$MERCURY_PERFORMANCE_DATABASE_URL" in
    */"$performance_database"|*/"$performance_database"\?*) ;;
    *)
        echo "MERCURY_PERFORMANCE_DATABASE_URL must target $performance_database" >&2
        exit 1
        ;;
esac

docker compose exec -T postgres dropdb --if-exists --force -U mercury "$performance_database"
docker compose exec -T postgres createdb -U mercury "$performance_database"

go run -tags="$GOOSE_TAGS" "github.com/pressly/goose/v3/cmd/goose@$GOOSE_VERSION" \
    -dir migrations postgres "$MERCURY_PERFORMANCE_DATABASE_URL" up

docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U mercury -d "$performance_database" \
    -f /dev/stdin < testdata/performance/seed.sql

counts=$(docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -tAc \
    "WITH cohorts(country_code) AS (VALUES ('XA'),('XB'),('XC'),('XD'),('XE')), actual AS (SELECT t.country_code,count(*) FILTER (WHERE c.state='ACTIVE' AND c.placement_code='search_results') AS eligible FROM campaign_target_countries t JOIN campaigns c ON c.id=t.campaign_id GROUP BY t.country_code) SELECT string_agg(cohorts.country_code || ':' || COALESCE(actual.eligible,0),',' ORDER BY cohorts.country_code) FROM cohorts LEFT JOIN actual USING (country_code)" \
    -U mercury -d "$performance_database")
[ "$counts" = "XA:0,XB:1,XC:10,XD:100,XE:1000" ] || { echo "unexpected performance cohort counts: $counts" >&2; exit 1; }
echo "$counts"
