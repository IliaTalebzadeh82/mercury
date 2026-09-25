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
