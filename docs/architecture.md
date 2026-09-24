# Phase 0 architecture

## Runtime topology

Mercury currently has three runtime components:

```text
Browser -> Next.js operations shell -> Go HTTP application -> PostgreSQL
```

The Next.js server reads the backend operational endpoints. It converts a
backend readiness response into one of four presentation states: healthy,
degraded, unavailable, or frontend-misconfigured. It contains no authoritative
business logic.

The Go application is one deployable modular monolith. `cmd/mercury` is the
composition root. Code under `internal/platform` owns only configuration,
database connectivity, and HTTP lifecycle concerns. Domain packages will be
introduced only when their behavior exists.

PostgreSQL is the sole datastore. Phase 0 does not create a production schema.

## Startup

The process parses and validates configuration before opening its listener. It
constructs a `pgxpool` and performs a bounded PostgreSQL ping. Missing required
configuration, malformed configuration, or failed initial connectivity causes
startup to fail with a non-zero exit status.

The Phase 0 pool is explicitly capped at four connections. This conservative
limit prevents CPU-derived defaults from creating an unexpectedly large
database connection budget. Capacity will be revisited when real transactional
workloads exist.

## Liveness and readiness

`/healthz` answers whether the HTTP process is alive. It is independent of
PostgreSQL so dependency failure does not cause a supervisor to restart a
healthy process repeatedly.

`/readyz` first checks the application's readiness flag, then performs a
bounded real PostgreSQL ping. It returns:

- HTTP 200 and `ready` when work can be accepted.
- HTTP 503 and `database_unavailable` when PostgreSQL cannot be reached.
- HTTP 503 and `not_ready` whenever the internal lifecycle state has withdrawn
  readiness. During normal shutdown, listener closure begins immediately, so
  this is not a load-balancer drain guarantee.

Phase 0 has no useful backend behavior without PostgreSQL, so readiness fails
closed. A future serving data plane may require a different degradation policy;
that decision belongs to the phase that introduces projected serving state.

## Shutdown

SIGINT or SIGTERM causes the server shutdown path to withdraw readiness and
immediately start bounded `net/http` graceful shutdown. The listener stops
accepting work, active requests receive the configured shutdown window, the
PostgreSQL pool closes, and the process exits.

Phase 0 does not implement a load-balancer drain interval. Readiness withdrawal
records the application lifecycle state, but the listener begins closing
immediately. Externally observable drain behavior will be designed only when an
actual orchestrator or load balancer exists.

## Migration boundary

Production migrations belong in `migrations/`, alongside the production state
they introduce. Phase 0 has no production state, so that directory contains no
SQL migration. Goose mechanics are verified through a reversible fixture in
`testdata/migrations`, which is never a production migration source. The
fixture runs only against the disposable `mercury_migration_test` database;
that database is reset before and after verification so its Goose metadata can
never conflict with the application database.

## Deliberately absent

There are no campaign, budget, serving, event, measurement, attribution,
analytics, cache, or messaging abstractions. Kafka, Redis, ClickHouse,
Kubernetes, Helm, Terraform, gRPC, GraphQL, ORMs, Redux, and ML are outside the
Phase 0 boundary.
