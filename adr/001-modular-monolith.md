# ADR 001: Begin as a modular monolith

## Status

Accepted for Phase 0.

## Context

Mercury will eventually explore control-plane, latency-sensitive serving,
measurement, accounting, analytics, and reconciliation concerns. None of those
workloads exists yet. Splitting deployables now would create network failure,
versioning, deployment, and data-ownership problems without solving an observed
scaling or availability constraint.

## Decision

The backend begins as one Go executable with cohesive packages inside one
repository. Package boundaries follow behavior that exists. Process separation
is not inferred from directory boundaries.

## Alternatives considered

- Start with independently deployed services for anticipated domains.
- Put all code in one undifferentiated package.
- Begin with a modular monolith and extract only under demonstrated pressure.

## Tradeoffs

The monolith keeps calls and transactions local, makes integration work small,
and permits boundaries to follow learned domain semantics. It does not provide
independent scaling, deployment, or failure isolation. Poor package discipline
could also create internal coupling that later extraction exposes.

## Consequences

Every proposed service extraction requires a separate ADR explaining the
concrete constraint, ownership and data changes, communication semantics, new
failure modes, and operational cost. Until then, modules share one process but
must retain clear ownership.

## Revisit conditions

Revisit when a boundary demonstrably requires independent scaling,
availability, deployment cadence, ownership, data isolation, latency, or
failure isolation.
