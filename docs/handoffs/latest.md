# Latest handoff

## Completed

- Phase 0 — Engineering Foundation: **COMPLETE**
- Phase 1 — Campaign Control Plane: **COMPLETE**
- Phase 2 — Deterministic Ad Decision Engine: **COMPLETE**
- Phase 3 — Strict Budget Accounting Baseline: **COMPLETE**

## Current baseline

Mercury is a Go modular monolith with PostgreSQL authority, a deterministic direct-PostgreSQL decision path, serialized immediate budget accounting, and an existing product/demo Next.js frontend. Phase 0–3 runtime semantics are unchanged by the roadmap migration.

Multi-instance safety is a global invariant for future evolution. Application correctness under replication must be proved before Kubernetes automates replication.

Not yet present: Kafka, Redis, campaign propagation, serving projections, a separate serving data plane, frequency caps, pacing, distributed budget allocation, Kubernetes, or multi-region behavior.

## Next phase

Phase 4 — Performance Baseline & Bottleneck Characterization

**Status: NOT STARTED.** It requires explicit user authorization in a fresh conversation. Do not create its ExecPlan before authorization.

## Durable context

- [Repository instructions](../../AGENTS.md)
- [Roadmap V2](../roadmap.md)
- [Phase 4 specification](../phases/phase-04.md)
- [Context map](../context-map.md)
- [Context lifecycle](../context-lifecycle.md)
- [Architecture](../architecture.md)
- [Invariants](../invariants.md)
- [Portfolio boundary](../portfolio-boundary.md)
