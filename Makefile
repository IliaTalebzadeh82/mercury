GO_CACHE ?= /tmp/mercury-go-cache
GO_MOD_CACHE ?= /tmp/mercury-go-mod
GOOSE_VERSION := v3.24.1
GOOSE_TAGS := no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb

export MERCURY_TEST_DATABASE_URL
export MERCURY_MIGRATION_TEST_DATABASE_URL

.PHONY: fmt-check vet unit-test integration-test test-race build backend-check infra-up infra-down migrate-test frontend-install frontend-lint frontend-test frontend-build check

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -d cmd internal && exit 1)

vet:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go vet ./...

unit-test:
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -count=1 ./...

integration-test:
	@test -n "$$MERCURY_TEST_DATABASE_URL" || { echo "MERCURY_TEST_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -tags=integration -count=1 ./internal/platform/database ./internal/platform/server

test-race:
	@test -n "$$MERCURY_TEST_DATABASE_URL" || { echo "MERCURY_TEST_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -race -count=1 ./...
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -race -tags=integration -count=1 ./internal/platform/database ./internal/platform/server

build:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go build -o bin/mercury ./cmd/mercury

backend-check: fmt-check vet unit-test integration-test test-race build

infra-up:
	docker compose up -d --wait postgres

infra-down:
	docker compose down

migrate-test:
	@test -n "$$MERCURY_MIGRATION_TEST_DATABASE_URL" || { echo "MERCURY_MIGRATION_TEST_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOOSE_VERSION=$(GOOSE_VERSION) GOOSE_TAGS='$(GOOSE_TAGS)' sh testdata/migrations/verify.sh

frontend-install:
	cd web && pnpm install --frozen-lockfile

frontend-lint:
	cd web && pnpm lint

frontend-test:
	cd web && pnpm test

frontend-build:
	cd web && pnpm build

check: backend-check migrate-test frontend-lint frontend-test frontend-build
