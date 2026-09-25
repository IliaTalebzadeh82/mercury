# ADR 005: Direct PostgreSQL decision consistency and read path

## Status

Accepted for Phase 2.

## Context

The first decision engine must select against authoritative campaign lifecycle,
placement, and country targeting without inventing a serving projection. A
multi-query read could mix committed states. Locks or stronger isolation would
add coordination to a path that only needs a statement-relative answer.

## Decision

The normal decision path uses one direct PostgreSQL statement at the default
`READ COMMITTED` isolation level. That statement resolves the placement,
filters all eligible campaigns, ranks every eligible candidate, returns the
exact candidate count, and distinguishes an unsupported placement from a valid
no-fill. It takes no row locks and starts no explicit transaction.

A diagnostic evaluation is deliberately different: selection and its bounded
explanations run in one read-only `REPEATABLE READ` transaction so both describe
the same snapshot. Diagnostics are configuration-gated and are not the serving
path.

## Tradeoffs

This keeps the first path simple and gives an exact committed statement
snapshot. A campaign observed active may be paused after the snapshot and
before the response; that does not invalidate the completed decision. Every
request computes ranking in PostgreSQL, so latency grows with the eligible
candidate set and PostgreSQL availability directly affects decision
availability.

## Consequences

Campaign pause committed before a decision statement excludes the campaign.
Pause committed after the statement snapshot does not rewrite its result.
Database failure is an error, never a no-fill. Query plans and candidate-count
cost are measured before adding caches, replicas, or projections.

## Revisit conditions

Revisit when measured latency, throughput, availability, or propagation needs
cannot be met by the direct authoritative read. Any serving projection must
define freshness, authority, failure, and reconciliation semantics separately.
