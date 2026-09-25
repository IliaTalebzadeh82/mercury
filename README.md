# Mercury

Mercury is an experimental real-time advertising platform exploring how
low-latency serving systems make economically correct decisions under
concurrent budgets, asynchronous state, high-volume event streams, and partial
failure.

The project is a modular monolith. Phase 1 adds a transactional campaign control
plane for advertisers, platform placements, configured lifetime budgets,
country targeting, lifecycle commands, idempotent creation, and optimistic
operator concurrency. It intentionally contains no serving, spend, reservation,
pacing, event, attribution, or analytics model.

Mercury is inspired by publicly discussed engineering problems in large-scale
advertising marketplaces. It is not associated with, and does not claim to
reproduce, Delivery Hero or any other company's proprietary architecture.

## Prerequisites

- Go 1.23.5
- Docker with Docker Compose
- Node.js 20.9 or later
- pnpm 10.17.1

The host used for Phase 0 verification has Node.js 18, so frontend checks were
run in `node:22-alpine`. CI also uses Node.js 22.

## Start the foundation

Start PostgreSQL:

```sh
make infra-up
```

Start the backend:

```sh
MERCURY_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury?sslmode=disable' \
  go run ./cmd/mercury
```

Start the frontend in another terminal:

```sh
cd web
MERCURY_API_URL='http://localhost:8080' pnpm dev
```

The frontend is available at `http://localhost:3000`. The backend listens at
`http://localhost:8080` by default.

## Operational endpoints

- `GET /healthz` reports process liveness and deliberately does not depend on
  PostgreSQL.
- `GET /readyz` reports whether the process is accepting work and PostgreSQL is
  reachable within the configured timeout. Database loss returns HTTP 503 with
  `{"status":"database_unavailable"}`.

The frontend renders readiness failure as degraded platform state. It does not
turn a backend dependency failure into an application-rendering failure.
Malformed explicit frontend API configuration is shown separately as a
misconfigured state.

## Configuration

| Variable | Required | Default | Purpose |
| --- | --- | --- | --- |
| `MERCURY_DATABASE_URL` | yes | none | Backend PostgreSQL connection string |
| `MERCURY_HTTP_ADDRESS` | no | `:8080` | Backend listen address |
| `MERCURY_DATABASE_PING_TIMEOUT` | no | `2s` | Startup and readiness database timeout |
| `MERCURY_DATABASE_OPERATION_TIMEOUT` | no | `3s` | Normal application database-operation deadline |
| `MERCURY_SHUTDOWN_TIMEOUT` | no | `10s` | Graceful HTTP shutdown deadline |
| `MERCURY_LOG_LEVEL` | no | `info` | `slog` level |
| `MERCURY_TEST_DATABASE_URL` | integration tests | none | Real PostgreSQL used by backend tests |
| `MERCURY_MIGRATION_TEST_DATABASE_URL` | migration verification | none | Dedicated disposable Goose verification database |
| `MERCURY_API_URL` | frontend server | `http://localhost:8080` | Backend base URL |

See `.env.example` for local values. Never commit real credentials.

## Verification

With PostgreSQL running, the complete local gate is:

```sh
MERCURY_TEST_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury_integration_test?sslmode=disable' \
MERCURY_MIGRATION_TEST_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury_migration_test?sslmode=disable' \
  make check
```

Backend checks are explicitly separated into `make unit-test` and
`make integration-test`. The integration target requires a real PostgreSQL
instance and fails when its database URL is absent.

Frontend checks:

```sh
cd web
pnpm install --frozen-lockfile
pnpm lint
pnpm test
pnpm build
pnpm exec playwright install chromium
pnpm e2e
```

`make migrate-test` resets the dedicated `mercury_migration_test` database,
applies the production migration, verifies placement seeds and deferred country
cardinality, rolls back and verifies cleanup, reapplies, then resets the
database. Integration tests recreate `mercury_integration_test`. Neither shares
Goose metadata or destructive setup with the normal `mercury` database.

## Campaign API

The Phase 1 API is mounted at `/v1`. Creation requires `Idempotency-Key`.
Campaign mutations use the current strong ETag as `If-Match`; missing, malformed,
and stale preconditions return 428, 400, and 412 respectively. See
[`docs/domain-model.md`](docs/domain-model.md) and
[`docs/invariants.md`](docs/invariants.md) for the exact semantics.

## Architecture records

- [ADR 001: Begin as a modular monolith](adr/001-modular-monolith.md)
- [ADR 002: PostgreSQL is the transactional source of truth](adr/002-postgresql-source-of-truth.md)
- [ADR 003: Control-plane concurrency and retry semantics](adr/003-control-plane-concurrency-and-retries.md)
- [ADR 004: Typed relational country targeting](adr/004-typed-relational-country-targeting.md)

The current topology and operational contracts are described in
[docs/architecture.md](docs/architecture.md).
