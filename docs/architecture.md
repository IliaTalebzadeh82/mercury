# Architecture through Phase 1

## Runtime topology

Mercury currently has three runtime components:

```text
Browser -> Next.js operations shell -> Go HTTP application -> PostgreSQL
```

The Next.js server reads the backend operational and campaign-control APIs. It converts a
backend readiness response into one of four presentation states: healthy,
degraded, unavailable, or frontend-misconfigured. It contains no authoritative
business logic.

The Go application remains one deployable modular monolith. `cmd/mercury` is the
composition root. Code under `internal/platform` owns only configuration,
database connectivity, and HTTP lifecycle concerns. Domain packages will be
introduced only when their behavior exists.

PostgreSQL is the sole datastore. Phase 1 adds advertiser and campaign
transactional state without creating a separate service or datastore.

## Campaign control plane

`internal/advertiser` owns advertiser creation and lookup.
`internal/campaign` owns campaign configuration, lifecycle, targeting, and
PostgreSQL transactions. `internal/api` adapts those commands to HTTP without
owning domain rules. The platform server mounts the API as an `http.Handler` and
does not import either domain package.

The browser continues to use the Next.js server boundary. Backend topology and
`MERCURY_API_URL` are not exposed through public client configuration.

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

Production migrations belong in `migrations/`. Their up/down/reapply behavior
runs only against `mercury_migration_test`. Real domain and concurrency tests
run against separately recreated `mercury_integration_test`; neither workflow
destructively initializes the normal `mercury` development database.

## Deliberately absent

There are no serving, spend, reservation, pacing, event, measurement,
attribution, analytics, cache, or messaging abstractions. Kafka, Redis, ClickHouse,
Kubernetes, Helm, Terraform, gRPC, GraphQL, ORMs, Redux, and ML remain outside
the Phase 1 boundary.
