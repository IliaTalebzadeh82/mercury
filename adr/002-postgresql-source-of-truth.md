# ADR 002: PostgreSQL is the transactional source of truth

## Status

Accepted for Phase 0.

## Context

Future Mercury state will include lifecycle transitions and financial
invariants that require explicit constraints, transactions, isolation, and
concurrency control. Adding multiple authoritative datastores would make those
guarantees and their recovery semantics harder to understand before any
workload justifies that cost.

## Decision

PostgreSQL is the sole authoritative transactional datastore. The backend uses
`pgx` and explicit SQL. Schema changes use Goose. Phase 0 validates the
connection and tooling but creates no production domain schema.

## Alternatives considered

- An in-memory starting point followed by later persistence.
- An ORM over PostgreSQL.
- Separate databases selected in advance for anticipated domains.
- PostgreSQL with explicit SQL as the single initial authority.

## Tradeoffs

PostgreSQL exposes the transaction, lock, constraint, and query-plan behavior
Mercury needs to study. Direct database access may eventually be too expensive
for the serving path, and a single authority can become an availability or
latency constraint. Those are measurable constraints rather than Phase 0
assumptions.

## Consequences

Transactional invariants belong in explicit application logic, database
constraints, and deliberately chosen transaction boundaries. Future caches,
projections, analytical stores, and streams are subordinate copies unless a
later ADR deliberately changes ownership.

## Revisit conditions

Revisit when measured workload, availability, isolation, or analytical needs
cannot be met responsibly by PostgreSQL. Introducing another datastore must
define authority, consistency, recovery, and reconciliation semantics.
