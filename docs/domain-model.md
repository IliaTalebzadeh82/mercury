# Domain model through Phase 2

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
There is no spend, remaining amount, reservation, pacing, or currency conversion.

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
