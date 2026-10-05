# Architecture through Phase 3

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

PostgreSQL is the sole datastore. Phase 2 reads advertiser and campaign
transactional state directly for decisions without creating a separate service,
projection, cache, or datastore.

## Campaign control plane

`internal/advertiser` owns advertiser creation and lookup.
`internal/campaign` owns campaign configuration, lifecycle, targeting, and
PostgreSQL transactions. `internal/api` adapts those commands to HTTP without
owning domain rules. The platform server mounts the API as an `http.Handler` and
does not import either domain package.

The browser continues to use the Next.js server boundary. Backend topology and
`MERCURY_API_URL` are not exposed through public client configuration.

`internal/decision` owns opportunity normalization, deterministic ranking, the
single-statement production decision query, transient decision identity, and
bounded diagnostics. `internal/api` only adapts that behavior and conditionally
mounts the diagnostic endpoint. The frontend Decision Lab reaches both through
server actions; no backend URL or diagnostic configuration becomes browser
configuration.

`internal/budget` owns authoritative budget snapshots, immediate consumption,
financial idempotency, and the PostgreSQL transaction that atomically updates
committed spend and appends an immutable receipt. It uses the campaign row as
the per-campaign serialization point shared with campaign lifecycle and budget
configuration commands. It does not own campaign lifecycle or decision
ranking.

The browser reads accounting through the Next.js server boundary. One logical
consumption attempt retains one idempotency key across an ambiguous 503. The UI
never calculates or optimistically increments remaining budget.

## Startup

The process parses and validates configuration before opening its listener. It
constructs a `pgxpool` and performs a bounded PostgreSQL ping. Missing required
configuration, malformed configuration, or failed initial connectivity causes
startup to fail with a non-zero exit status.

The PostgreSQL pool remains explicitly capped at four connections. This conservative
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

### Future replicated-workload semantics

Liveness answers: **is this process alive and functioning?** External dependency
failure must not automatically mark a functioning process dead and trigger a
restart loop.

Readiness answers: **is this particular instance currently safe to receive its
intended workload?** Future serving readiness may depend on projection bootstrap
completion, freshness within an explicit bound, required local indexes, required
ownership where applicable, and not being in drain state. A process can be alive
while unready. Process existence must never be conflated with safe serving.

Every serving replica must establish readiness from its own reconstructable state
and durable inputs; it cannot depend on a particular sibling remaining alive.

## Shutdown

SIGINT or SIGTERM causes the server shutdown path to withdraw readiness and
immediately start bounded `net/http` graceful shutdown. The listener stops
accepting work, active requests receive the configured shutdown window, the
PostgreSQL pool closes, and the process exits.

Phase 0 does not implement a load-balancer drain interval. Readiness withdrawal
records the application lifecycle state, but the listener begins closing
immediately. Externally observable drain behavior will be designed only when an
actual orchestrator or load balancer exists.

Future horizontally replicated workloads generally withdraw readiness, stop new
routing, boundedly finish or terminate in-flight work according to their
contract, release background ownership, close resources, and exit within their
termination deadline. The precise loss/duplication guarantee belongs to each
workflow and must be proved rather than assumed.

## Future state and replication conventions

Meaningful runtime state is classified by restart and replication semantics:

| Class | Meaning | Example |
| --- | --- | --- |
| `AUTHORITATIVE_SHARED` | Shared source of truth | PostgreSQL campaign state |
| `DURABLE_PARTITIONED` | Durable state under explicit partition ownership | possible future regional budget allocation |
| `DERIVED_RECONSTRUCTABLE` | Rebuildable from durable authority/history | future serving projection |
| `EPHEMERAL_INSTANCE_LOCAL` | Disposable instance/request state | request scratch data |

Process-local state cannot silently become cross-instance authority. Future
APIs, events, projection formats, database changes, and configuration also need
N/N+1 compatibility because rolling deployments are not atomic. Kubernetes
comes only after application semantics are proven under independent processes;
it orchestrates correct replicas rather than creating correctness.

## Migration boundary

Production migrations belong in `migrations/`. Their up/down/reapply behavior
runs only against `mercury_migration_test`. Real domain and concurrency tests
run against separately recreated `mercury_integration_test`; neither workflow
destructively initializes the normal `mercury` development database.

## Deliberately absent

There are no serving-projection, reservation, pacing, event,
measurement, attribution, analytics, cache,
or messaging abstractions. Phase 3 spend is immediate; it has no pending,
release, expiration, or late-charge model. Kafka, Redis, ClickHouse,
Kubernetes, Helm, Terraform, gRPC, GraphQL, ORMs, Redux, and ML remain outside
the Phase 3 boundary.
