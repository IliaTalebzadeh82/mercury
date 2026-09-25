# ADR 007: Immediate consumption boundary and accounting representation

## Status

Accepted for Phase 3.

## Context

Mercury needs to prove concurrent budget correctness before it has impressions,
delivery confirmation, or reservations. A Phase 2 decision is transient and is
not evidence that an ad was delivered or that money is billable. Accounting
must also be reconstructable without making every balance check aggregate an
ever-growing ledger.

## Decision

Phase 3 exposes an independent immediate budget-consumption command. The caller
declares an amount billable now. The command does not mean that an ad was
rendered, an impression occurred, or money was reserved.

PostgreSQL stores an immutable command/result receipt for every completed
business outcome. Approved receipts are the reconstructable spend history.
`campaigns.committed_spend_minor` is the transactionally maintained enforcement
aggregate. Remaining budget is always derived as configured budget minus
committed spend. An approved receipt and its counter change are atomic.

## Alternatives considered

- Charge as part of the Phase 2 decision.
- Introduce an authorization token or reservation.
- Keep only a counter.
- Calculate the current balance from the ledger for every command.

## Tradeoffs

The decision path can select a campaign that subsequently returns
`INSUFFICIENT_BUDGET`. This gap is intentional and visible. The aggregate makes
enforcement constant-time but creates a ledger/counter equality invariant that
is maintained by the sanctioned transaction and verified rather than expressed
as a declarative cross-table constraint.

## Consequences

Phase 2 remains budget-unaware. Phase 3 has no late-charge semantics, pending
state, expiration, or release. Approved ledger sum and the counter must be
checked after concurrency and load fixtures.

## Revisit conditions

Revisit when Phase 4 introduces a concrete reserve/serve/confirm/commit
lifecycle or when measured ledger/counter maintenance requires a different
accounting representation.
