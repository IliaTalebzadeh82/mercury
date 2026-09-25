# Mercury invariants

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
- Every actual mutation increments campaign version exactly once. Semantic
  no-ops leave the version unchanged.
- A stale If-Match version cannot overwrite newer campaign state.
- Retried creation with the same normalized command returns the original
  resource. Reusing its key for different input is a conflict.

Database checks, foreign keys, uniqueness constraints, serialized target
removals, and deferred country-cardinality triggers protect structural
invariants. Domain commands enforce transition and editability rules while
holding the campaign row lock.
