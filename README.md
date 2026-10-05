# Mercury

Mercury is an experimental real-time advertising platform exploring how
low-latency serving systems make economically correct decisions under
concurrent budgets, asynchronous state, high-volume event streams, and partial
failure.

The project is a modular monolith. Phase 1 supplies a transactional campaign
control plane. Phase 2 adds a deliberately small direct-PostgreSQL decision
engine with exact eligibility, deterministic rendezvous-style selection,
transient evaluation identity, bounded optional diagnostics, and a developer
Decision Lab. Phase 3 adds strictly serialized immediate budget consumption,
an immutable accounting ledger, financial idempotency, and explicit
counter/ledger verification. It intentionally contains no reservation, pacing,
event, attribution, analytics, or serving-projection model.

Mercury is inspired by publicly discussed engineering problems in large-scale
advertising marketplaces. It is not associated with, and does not claim to
reproduce, Delivery Hero or any other company's proprietary architecture.

## Project direction and working context

The completed Phase 0–3 implementation is Mercury's correctness baseline. Once
authorized, the next phase will measure that baseline before introducing
architectural complexity. See the [Roadmap V2](docs/roadmap.md),
[portfolio boundary](docs/portfolio-boundary.md), and
[company-evidence policy](docs/company-evidence.md).

Repository-guided development starts with [AGENTS.md](AGENTS.md), the
[latest handoff](docs/handoffs/latest.md), and the current phase specification.
The [context map](docs/context-map.md) and
[context lifecycle](docs/context-lifecycle.md) keep future sessions narrow and
resumable. The original one-file project prompt is preserved only as
[historical audit material](docs/archive/masterprompt-original.md).

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
| `MERCURY_DIAGNOSTIC_API_ENABLED` | no | `false` | Mount the bounded read-only decision explanation endpoint |
| `MERCURY_TEST_DATABASE_URL` | integration tests | none | Real PostgreSQL used by backend tests |
| `MERCURY_MIGRATION_TEST_DATABASE_URL` | migration verification | none | Dedicated disposable Goose verification database |
| `MERCURY_PERFORMANCE_DATABASE_URL` | performance fixture | none | Dedicated disposable Phase 2 baseline database |
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
applies the production migrations, verifies placement seeds, deferred country
cardinality, the Phase 2 index, and the Phase 3 accounting schema, rolls Phase 3 down and up, rolls all
migrations down and verifies cleanup, reapplies, then resets the
database. Integration tests recreate `mercury_integration_test`. Neither shares
Goose metadata or destructive setup with the normal `mercury` database.

## Campaign API

The Phase 1 API is mounted at `/v1`. Creation requires `Idempotency-Key`.
Campaign mutations use the current strong ETag as `If-Match`; missing, malformed,
and stale preconditions return 428, 400, and 412 respectively. See
[`docs/domain-model.md`](docs/domain-model.md) and
[`docs/invariants.md`](docs/invariants.md) for the exact semantics.

## Decision API

`POST /v1/ad-decisions` accepts strict JSON containing `opportunity_id`,
`placement`, and `country`. Fill and no-fill both return HTTP 200. The normal
response exposes only normalized opportunity fields, outcome, transient
decision ID, and the selected campaign ID/version when filled.

`POST /v1/ad-decisions/explain` is not mounted unless
`MERCURY_DIAGNOSTIC_API_ENABLED=true`. It returns at most 200 campaign
explanations and is developer diagnostics, not an authentication boundary.
See [`docs/domain-model.md`](docs/domain-model.md) for ranking and identity
semantics.

The reproducible load fixture, k6 workload, query-plan evidence, and measured
development baseline are documented in
[`docs/performance/phase2-baseline.md`](docs/performance/phase2-baseline.md).

## Budget accounting API

`GET /v1/campaigns/{campaign_id}/budget` returns configured budget, committed
spend, and derived remaining budget from one authoritative PostgreSQL row
snapshot.

`POST /v1/campaigns/{campaign_id}/budget-consumptions` requires an
`Idempotency-Key` and a positive decimal-string `amount_minor` plus currency.
It returns `APPROVED`, `INSUFFICIENT_BUDGET`, or `CAMPAIGN_NOT_ACTIVE` as an
HTTP 200 business result. A 503 is infrastructure uncertainty; callers must
retry the same semantic command with the same key.

Immediate consumption does not assert delivery and is not a reservation.
`ACTIVE` does not guarantee that remaining budget is positive. Phase 2 remains
budget-unaware by design.

The reproducible Phase 3 contention matrix and measured development baseline
are documented in
[`docs/performance/phase3-baseline.md`](docs/performance/phase3-baseline.md).

## Architecture records

- [ADR 001: Begin as a modular monolith](adr/001-modular-monolith.md)
- [ADR 002: PostgreSQL is the transactional source of truth](adr/002-postgresql-source-of-truth.md)
- [ADR 003: Control-plane concurrency and retry semantics](adr/003-control-plane-concurrency-and-retries.md)
- [ADR 004: Typed relational country targeting](adr/004-typed-relational-country-targeting.md)
- [ADR 005: Direct PostgreSQL decision consistency and read path](adr/005-direct-postgresql-decision-read-path.md)
- [ADR 006: Opportunity and decision identity](adr/006-opportunity-and-decision-identity.md)
- [ADR 007: Immediate consumption boundary and accounting representation](adr/007-immediate-consumption-and-accounting-representation.md)
- [ADR 008: PostgreSQL budget serialization](adr/008-postgresql-budget-serialization.md)
- [ADR 009: Financial idempotency](adr/009-financial-idempotency.md)

The current topology and operational contracts are described in
[docs/architecture.md](docs/architecture.md).
