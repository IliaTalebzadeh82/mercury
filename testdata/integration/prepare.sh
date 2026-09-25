#!/bin/sh
set -eu

integration_database=mercury_integration_test

case "$MERCURY_TEST_DATABASE_URL" in
    */"$integration_database"|*/"$integration_database"\?*) ;;
    *)
        echo "MERCURY_TEST_DATABASE_URL must target $integration_database" >&2
        exit 1
        ;;
esac

docker compose exec -T postgres dropdb --if-exists --force -U mercury "$integration_database"
docker compose exec -T postgres createdb -U mercury "$integration_database"

go run -tags="$GOOSE_TAGS" "github.com/pressly/goose/v3/cmd/goose@$GOOSE_VERSION" \
    -dir migrations postgres "$MERCURY_TEST_DATABASE_URL" up
