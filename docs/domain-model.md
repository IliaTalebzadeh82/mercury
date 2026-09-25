# Domain model through Phase 3

## Advertiser

An advertiser owns campaigns. Phase 1 stores only its UUID, display name, and
creation metadata. It is not a user, billing account, or legal/KYC entity and
has no lifecycle or currency.

## Placement

Placements are Mercury-owned inventory surfaces. The read-only Phase 1 catalog
contains `home_feed`, `restaurant_list`, and `search_results`. The stable code is
the placement identifier.

## Campaign

A campaign belongs to one advertiser and is complete when created. `DRAFT`
means valid but not enabled, never partially configured. A campaign has one
placement, one immutable currency, one positive lifetime configured budget,
and at least one country target.

Money is an integer count of minor units. Phase 1 supports EUR, GBP, and USD.
There is no reservation, pacing, or currency conversion.

Country targeting is a normalized set of uppercase two-letter codes. Phase 1
validates their shape, not membership in a country-catalog subsystem.

## Lifecycle

```text
DRAFT  -> ACTIVE -> PAUSED -> ACTIVE
   \         \        \
    +---------+--------+--> ENDED
```

`ENDED` is terminal. `ACTIVE` means enabled in the transactional control plane;
it makes no serving, propagation, or funding claim.

Name is editable in every non-ended state. Active budgets may stay equal or
increase, never decrease. Placement and targeting can change only while draft
or paused. Currency never changes.

There is no scheduling, exhaustion, deletion, or archival in Phase 1.

## Ad opportunity and decision

An ad opportunity is the transient tuple of caller-supplied canonical UUID,
placement, and two-letter country code. It contains no user, device, search,
creative, bid, or personalization data and is never persisted.

A campaign is eligible exactly when it is `ACTIVE`, its placement matches, and
its country targets contain the normalized opportunity country. Configured
budget is intentionally irrelevant because Phase 2 has no spend or available
budget state.

Eligible campaigns use deterministic rendezvous-style ranking. For every
campaign, Mercury calculates the full MD5 byte sequence of:

```text
canonical opportunity UUID + ":" + canonical campaign UUID
```

The highest digest wins; equal digests use campaign UUID ascending. MD5 exists
only for identical PostgreSQL/Go non-security ordering and must never be reused
for passwords, tokens, signatures, or any security purpose.

The same opportunity and committed candidate set therefore produce the same
winner. Each successful evaluation—including no-fill—receives a fresh transient
random UUIDv4 `decision_id`. Neither opportunity nor decision is stored.

## Budget account and immediate consumption

Every campaign has one configured budget and a transactionally maintained
committed-spend aggregate in the same immutable currency. Current remaining
budget is derived, never independently stored:

```text
remaining budget = configured budget - committed spend
```

An immediate budget-consumption command asks Mercury to commit a positive
minor-unit amount now. It does not mean an ad was rendered, an impression
occurred, delivery was confirmed, or money was reserved. Only an `ACTIVE`
campaign can approve consumption. Completed outcomes are `APPROVED`,
`INSUFFICIENT_BUDGET`, and `CAMPAIGN_NOT_ACTIVE`; all three are immutable and
idempotently replayable.

`ACTIVE` continues to mean enabled in the control plane, not guaranteed
spendable. An active campaign may have zero remaining budget. Phase 2 can select
it, and a later independent consumption can return `INSUFFICIENT_BUDGET`. This
gap is intentional in Phase 3.

Approved consumption receipts are the reconstructable spend history.
`campaigns.committed_spend_minor` is the fast enforcement aggregate. Phase 3
contains no reservation, pending accounting state, late charge, expiration,
release, or automatic exhausted lifecycle.
