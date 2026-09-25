# ADR 009: Financial idempotency and durable outcome replay

## Status

Accepted for Phase 3.

## Context

A committed financial response can be lost, and a database connection can fail
while commit durability is uncertain. Retrying with a fresh identity could
double-charge. Allowing a completed rejection to change on retry would also
make caller behavior timing-dependent.

## Decision

Idempotency is scoped by `(campaign_id, idempotency_key)`. A SHA-256 fingerprint
covers canonical campaign ID, positive minor-unit amount, and normalized
currency; it excludes the raw key.

`APPROVED`, `INSUFFICIENT_BUDGET`, and `CAMPAIGN_NOT_ACTIVE` are persisted and
replayed. Reusing a key for a different command is a conflict. Malformed input,
currency mismatch, missing campaigns, and infrastructure failures are not
persisted as business outcomes.

If commit might have succeeded, Mercury returns `accounting_outcome_unknown`
and requires the same key. A retry either finds and replays the receipt or
executes after PostgreSQL rolled the earlier transaction back.

## Alternatives considered

- Idempotency only for approved charges.
- Process-memory deduplication.
- Caller retries with a fresh key after timeout.
- A pending command state.

## Tradeoffs

An insufficient or inactive outcome remains stable even if budget or lifecycle
later changes. Receipts retain opaque keys in PostgreSQL but keys and
fingerprints are never logged or exposed by the API.

## Consequences

The frontend retains one key across ambiguous failures and labels the action
`Retry safely`. A different key always means a deliberate new financial
attempt. There is no pending-state recovery worker.

## Revisit conditions

Revisit when authenticated actors, external billing events, retention policy,
or Phase 4 reservation identities introduce a broader financial command model.
