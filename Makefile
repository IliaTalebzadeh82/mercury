GO_CACHE ?= /tmp/mercury-go-cache
GO_MOD_CACHE ?= /tmp/mercury-go-mod
GOOSE_VERSION := v3.24.1
GOOSE_TAGS := no_clickhouse no_libsql no_mssql no_mysql no_sqlite3 no_vertica no_ydb

export MERCURY_TEST_DATABASE_URL
export MERCURY_MIGRATION_TEST_DATABASE_URL
export MERCURY_PERFORMANCE_DATABASE_URL

.PHONY: fmt-check vet unit-test integration-prepare integration-test test-race build backend-check infra-up infra-down migrate-up migrate-down migrate-test performance-seed frontend-install frontend-lint frontend-test frontend-build frontend-e2e check

fmt-check:
	@test -z "$$(gofmt -l cmd internal)" || (gofmt -d cmd internal && exit 1)

vet:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go vet ./...

unit-test:
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -count=1 ./...

integration-test:
	@test -n "$$MERCURY_TEST_DATABASE_URL" || { echo "MERCURY_TEST_DATABASE_URL is required"; exit 1; }
	@$(MAKE) integration-prepare
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -tags=integration -count=1 ./internal/advertiser ./internal/campaign ./internal/budget ./internal/decision ./internal/api ./internal/platform/database ./internal/platform/server

integration-prepare:
	@test -n "$$MERCURY_TEST_DATABASE_URL" || { echo "MERCURY_TEST_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOOSE_VERSION=$(GOOSE_VERSION) GOOSE_TAGS='$(GOOSE_TAGS)' sh testdata/integration/prepare.sh

test-race:
	@test -n "$$MERCURY_TEST_DATABASE_URL" || { echo "MERCURY_TEST_DATABASE_URL is required"; exit 1; }
	@$(MAKE) integration-prepare
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -race -count=1 ./...
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go test -race -tags=integration -count=1 ./internal/advertiser ./internal/campaign ./internal/budget ./internal/decision ./internal/api ./internal/platform/database ./internal/platform/server

build:
	GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go build -o bin/mercury ./cmd/mercury

backend-check: fmt-check vet unit-test integration-test test-race build

infra-up:
	docker compose up -d --wait postgres

infra-down:
	docker compose down

migrate-up:
	@test -n "$$MERCURY_DATABASE_URL" || { echo "MERCURY_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run -tags='$(GOOSE_TAGS)' github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir migrations postgres "$$MERCURY_DATABASE_URL" up

migrate-down:
	@test -n "$$MERCURY_DATABASE_URL" || { echo "MERCURY_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) go run -tags='$(GOOSE_TAGS)' github.com/pressly/goose/v3/cmd/goose@$(GOOSE_VERSION) -dir migrations postgres "$$MERCURY_DATABASE_URL" down

migrate-test:
	@test -n "$$MERCURY_MIGRATION_TEST_DATABASE_URL" || { echo "MERCURY_MIGRATION_TEST_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOOSE_VERSION=$(GOOSE_VERSION) GOOSE_TAGS='$(GOOSE_TAGS)' sh testdata/migrations/verify.sh

performance-seed:
	@test -n "$$MERCURY_PERFORMANCE_DATABASE_URL" || { echo "MERCURY_PERFORMANCE_DATABASE_URL is required"; exit 1; }
	@GOCACHE=$(GO_CACHE) GOMODCACHE=$(GO_MOD_CACHE) GOOSE_VERSION=$(GOOSE_VERSION) GOOSE_TAGS='$(GOOSE_TAGS)' sh testdata/performance/seed.sh

frontend-install:
	cd web && pnpm install --frozen-lockfile

frontend-lint:
	cd web && pnpm lint

frontend-test:
	cd web && pnpm test

frontend-build:
	cd web && pnpm build

frontend-e2e:
	@$(MAKE) integration-prepare
	cd web && pnpm e2e

check: backend-check migrate-test frontend-lint frontend-test frontend-build frontend-e2e
