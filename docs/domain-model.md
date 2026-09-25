# Phase 1 campaign control-plane domain

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
