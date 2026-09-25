# ADR 004: Typed relational country targeting

## Status

Accepted for Phase 1.

## Context

Phase 1 needs country targeting that PostgreSQL can validate and Phase 2 can
query directly. A flexible targeting document or generic rule engine would add
operator vocabularies, type interpretation, indexing choices, and validation
behavior before Mercury has more than one geographic dimension.

## Decision

Country targets are normalized uppercase two-letter codes stored in
`campaign_target_countries`. Placement is a direct campaign relationship, not a
generic targeting predicate. Every committed campaign must have at least one
country target; deferred constraint triggers enforce both campaign insertion
and removal of a final target while allowing atomic target replacement. An
immediate trigger locks the owning campaign before target removal so concurrent
transactions cannot both remove what each believes is a non-final row.

The Phase 1 contract validates code shape rather than introducing a country or
subdivision catalog.

## Alternatives considered

- A JSONB targeting document.
- Generic dimension/operator/value rows.
- Typed country and region tables.
- A typed country-only relation.

## Tradeoffs

The schema provides strong types, uniqueness, and transparent query behavior.
Adding a genuine future targeting dimension requires an intentional schema and
API change. Region hierarchy and actual ISO membership are not modeled.

## Consequences

Target replacement is transactional. Active campaigns cannot change targeting;
an operator pauses first. Phase 2 can use the same typed relation for initial
eligibility without parsing a rule language.

## Revisit conditions

Revisit when a concrete targeting dimension, region hierarchy, or inventory
query cannot be represented clearly with typed relational state.
