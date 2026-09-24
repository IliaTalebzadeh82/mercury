# Mercury

Mercury is an experimental real-time advertising platform exploring how
low-latency serving systems make economically correct decisions under
concurrent budgets, asynchronous state, high-volume event streams, and partial
failure.

The project begins as a modular monolith. Phase 0 contains only the engineering
foundation: a Go HTTP process, PostgreSQL connectivity, operational health
contracts, migration tooling, and a Next.js operations shell. It intentionally
contains no campaign, budget, serving, event, attribution, or analytics model.

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
| `MERCURY_SHUTDOWN_TIMEOUT` | no | `10s` | Graceful HTTP shutdown deadline |
| `MERCURY_LOG_LEVEL` | no | `info` | `slog` level |
| `MERCURY_TEST_DATABASE_URL` | integration tests | none | Real PostgreSQL used by backend tests |
| `MERCURY_MIGRATION_TEST_DATABASE_URL` | migration verification | none | Dedicated disposable Goose verification database |
| `MERCURY_API_URL` | frontend server | `http://localhost:8080` | Backend base URL |

See `.env.example` for local values. Never commit real credentials.

## Verification

With PostgreSQL running, the complete local gate is:

```sh
MERCURY_TEST_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury?sslmode=disable' \
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
```

`make migrate-test` resets the dedicated `mercury_migration_test` database,
applies the fixture under `testdata/migrations`, verifies version and table
state, rolls it back and verifies cleanup, reapplies and verifies it, then
resets the dedicated database again. It never shares Goose metadata with the
normal `mercury` database. The production `migrations/` set remains
intentionally empty until a real domain schema exists.

## Architecture records

- [ADR 001: Begin as a modular monolith](adr/001-modular-monolith.md)
- [ADR 002: PostgreSQL is the transactional source of truth](adr/002-postgresql-source-of-truth.md)

The current topology and operational contracts are described in
[docs/architecture.md](docs/architecture.md).
