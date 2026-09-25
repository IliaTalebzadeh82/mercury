# ADR 008: PostgreSQL serialization and strict no-overspend guarantee

## Status

Accepted for Phase 3.

## Context

Concurrent consumers, lifecycle commands, and configured-budget edits can all
change whether spend is legal. The guarantee must hold across any number of
Mercury processes without relying on process-local coordination.

## Decision

The authoritative campaign row is the per-campaign serialization point.
Consumption locks it with `SELECT ... FOR UPDATE` before reading lifecycle,
configured budget, or committed spend. Lifecycle and configured-budget
mutations use the same lock.

For one PostgreSQL primary, every committed state satisfies:

```text
0 <= committed_spend <= configured_budget
```

Affordability is checked as `amount <= configured - committed`; addition occurs
only after that subtraction proves it safe. PostgreSQL row constraints provide
the structural backstop.

## Alternatives considered

- Atomic conditional counter updates.
- Optimistic version retries.
- `SERIALIZABLE` transactions.
- Advisory locks.
- Process-local mutexes.

## Tradeoffs

Commands for one hot campaign serialize and can time out while waiting for the
row. Independent campaigns can proceed concurrently. Accounting writes also
create row churn on a table read by the direct Phase 2 decision path. Phase 3
measures both effects before tuning.

## Consequences

Database timeouts are infrastructure failures, never insufficient-budget
results. Accounting does not increment the configuration version or update the
configuration timestamp. All financial concurrency tests use real PostgreSQL
and inspect durable counter and ledger state.

## Revisit conditions

Revisit when measured contention, serving-read interference, regional
ownership, or a future reservation authority makes one campaign-row authority
insufficient.
