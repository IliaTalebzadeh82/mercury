# ADR 003: Control-plane concurrency and retry semantics

## Status

Accepted for Phase 1.

## Context

Campaign operators may submit concurrent lifecycle and configuration commands,
and creation responses can be lost after PostgreSQL commits. Silent lost updates
would make operator intent ambiguous, while storing a receipt for every command
would add retention and replay semantics that Phase 1 does not otherwise need.

## Decision

Advertiser and campaign creation require a persistent `Idempotency-Key`. The
resource row stores the key and a SHA-256 fingerprint of the validated,
normalized semantic command in the same transaction as creation. Equivalent
replays return the original resource; reuse for different input is a conflict.

Campaigns expose a monotonically increasing version through both JSON and a
strong HTTP ETag. Every existing-campaign mutation requires `If-Match`. A stale
version returns HTTP 412 and never overwrites current state. Transactions lock
the campaign row while checking authoritative state and applying a mutation.

Phase 1 does not store receipts for existing-resource commands. A retry after a
successful response was lost may therefore receive HTTP 412.

## Alternatives considered

- Last-write-wins updates protected only by database locking.
- Persistent receipts for every mutation.
- Optimistic versioning plus creation-only persistent idempotency.

## Tradeoffs

The chosen contract prevents silent overwrites and makes ambiguous creation
safe. Clients must handle 412 by reloading rather than retrying automatically.
It also leaves existing-resource response-loss ambiguity visible instead of
introducing a command-log subsystem prematurely.

## Consequences

Every state-changing campaign response increments the version exactly once;
semantic no-ops leave it unchanged. HTTP, frontend, transaction, and concurrency
tests must agree on 428, 412, and 409 semantics.

## Revisit conditions

Revisit when external automation needs guaranteed replay of mutation responses,
when command audit retention becomes a requirement, or when asynchronous
campaign propagation introduces a different version model.
