# ADR 006: Opportunity and decision identity

## Status

Accepted for Phase 2.

## Context

Deterministic ranking needs stable input, while each completed evaluation also
needs an identity. Treating those as one identifier would either make repeated
evaluations indistinguishable or accidentally promise idempotent decision
responses.

## Decision

`opportunity_id` is a caller-supplied canonical lowercase, non-nil UUID. It is
transient, is used as ranking input, and is echoed in successful responses.
Mercury does not persist it and does not use it as a metric label.

`decision_id` identifies one successfully completed evaluation. Mercury creates
a new random UUIDv4 backed by `crypto/rand` for every fill and no-fill. It is
also transient and is not retry-stable. Failed evaluations have no decision
representation or decision ID.

## Tradeoffs

Repeating an opportunity against an unchanged candidate set chooses the same
campaign but returns a different decision ID. Callers that require durable
deduplication or audit history do not receive it in Phase 2.

## Consequences

The two identifiers must remain semantically distinct in API responses, logs,
tests, and documentation. Neither introduces opportunity or decision storage.

## Revisit conditions

Revisit when a concrete event, accounting, attribution, auditing, or retry
contract requires durable identity and retention semantics.
