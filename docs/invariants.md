# Mercury invariants

## Global architecture invariants

### Multi-instance safety

Mercury assumes any independently deployable backend component may eventually run as several concurrent instances. No business correctness guarantee may depend on a singleton application process, process-local mutex/counter/durable state, sticky sessions, one scheduler, one consumer, or one process receiving every request for an entity unless that dependency is explicitly designed, bounded, documented, and verified.

Process-local state is acceptable only when it is derived or reconstructable, intentionally instance-scoped, non-authoritative for cross-instance correctness, and safe to lose on restart. Examples include local serving indexes, derived projections, ephemeral caches, request-local data, and bounded metrics buffers.

Cross-instance guarantees may use authoritative shared state, database constraints and transactions, idempotency, versioning, durable event streams, explicit partition ownership, justified leases, or explicitly bounded inconsistency. Distributed locking is not a default: first state the invariant and determine whether coordination is necessary.

Any phase adding mutable shared state, counters, projections/caches, workers, schedulers, consumers, leases, serving replicas, caps, pacing, or allocations must determine whether multiple independent processes change semantics. If they do, single-process tests are insufficient. Verify with the simplest realistic setup—multiple OS processes, Compose replicas, independent database connections/consumers, or local load balancing—before Kubernetes.

### Kubernetes boundary

Kubernetes does not make Mercury multi-instance safe. Application semantics must be correct under independent replicas before Kubernetes schedules, replaces, scales, routes, discovers, or rolls them out. Orchestration must not compensate for singleton assumptions, process-local correctness, unsafe shared state, broken ownership, or incorrect retries.

### Meaningful runtime state

State whose restart or replication semantics matter should be classified as one of:

- `AUTHORITATIVE_SHARED`: shared source of truth, such as current PostgreSQL campaign state.
- `DURABLE_PARTITIONED`: durable authority divided by explicit ownership, potentially future regional budget allocations.
- `DERIVED_RECONSTRUCTABLE`: rebuildable from durable authority/history, such as a future serving projection.
- `EPHEMERAL_INSTANCE_LOCAL`: disposable instance/request state, such as request scratch data.

Do not classify every variable. The classification exists to expose authority, loss, rebuild, and ownership semantics.

### Lifecycle and mixed versions

For future replicated workloads, shutdown must withdraw readiness, stop new routed work, boundedly complete or terminate in-flight work, relinquish background ownership safely, close resources, and exit within its deadline. Never claim zero loss without workflow-specific proof.

Changes to APIs, event schemas, projection formats, database schemas, or configuration must consider simultaneous N/N+1 operation. Use backward-compatible evolution, expand/migrate/contract, versioned events, or tolerant readers where appropriate; do not assume atomic deployment.

## Phase 1 campaign control plane

- Every campaign belongs to exactly one existing advertiser.
- Every committed campaign is complete and has a valid placement, positive
  lifetime configured budget, supported immutable currency, and at least one
  normalized country target.
- Currency is one of EUR, GBP, or USD. Mercury never combines money across
  currencies.
- Country targets are unique within a campaign.
- Lifecycle transitions are exactly DRAFT to ACTIVE or ENDED, ACTIVE to PAUSED
  or ENDED, and PAUSED to ACTIVE or ENDED. ENDED is terminal.
- Resume and activation revalidate authoritative persisted configuration.
- An active campaign's configured budget cannot decrease.
- Active placement and targeting are immutable; the operator pauses first.
- Every successful configuration or lifecycle mutation increments campaign
  version exactly once. Semantic no-ops and accounting consumption leave the
  configuration version unchanged.
- A stale If-Match version cannot overwrite newer campaign state.
- Retried creation with the same normalized command returns the original
  resource. Reusing its key for different input is a conflict.

Database checks, foreign keys, uniqueness constraints, serialized target
removals, and deferred country-cardinality triggers protect structural
invariants. Domain commands enforce transition and editability rules while
holding the campaign row lock.

## Phase 2 decision engine

- Eligibility is exactly active state, matching placement, and matching country target.
- Configured budget is neither read nor interpreted by decision eligibility.
- Every eligible candidate participates in ranking; the normal path is not capped.
- Ranking uses the full MD5 digest of canonical opportunity UUID, a colon, and
  canonical campaign UUID; highest digest wins and UUID ascending breaks ties.
- A valid placement with no eligible campaigns produces a successful no-fill.
- PostgreSQL failure cannot be converted to no-fill.
- One normal SQL statement supplies placement validity, eligibility, candidate
  count, and winner from one committed snapshot without locks.
- Successful repeated evaluations may share a winner but never a decision ID.
- Diagnostic explanations are limited to 200 campaigns, have fixed rejection
  reason order, include the selected campaign, and share one repeatable-read
  snapshot with their selection.

## Phase 3 concurrent budget accounting

- Configured budget, committed spend, and consumption amounts use integer minor
  units in the campaign's immutable currency.
- Every committed campaign row satisfies
  `0 <= committed_spend <= configured_budget`.
- Remaining budget is derived as configured budget minus committed spend.
- Consumption locks the campaign row before evaluating lifecycle or funds.
- Affordability uses `amount <= configured - committed`; addition happens only
  after that comparison proves it cannot overflow.
- `ACTIVE` consumption completes as `APPROVED` or `INSUFFICIENT_BUDGET`.
  Draft, paused, and ended consumption completes as `CAMPAIGN_NOT_ACTIVE`.
- Every completed business result is immutable and replayable by its
  campaign-scoped idempotency key and semantic fingerprint.
- Approved counter change and receipt insertion commit atomically.
- Approved receipt sum equals the committed-spend aggregate through the
  sanctioned transaction path. This is verified, not a declarative cross-table
  constraint.
- Configured budget can never fall below committed spend. Existing active and
  ended mutation rules continue to apply.
- Accounting does not change campaign configuration version or `updated_at`.
- `ACTIVE` does not imply positive remaining budget, and Phase 2 remains
  budget-unaware.
