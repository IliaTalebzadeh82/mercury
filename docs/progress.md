# Mercury progress history

This is historical evidence, not default startup context. Commit descriptions below were verified from repository history; no test or benchmark counts are inferred.

| Phase | Commit boundary | Preserved result |
| --- | --- | --- |
| 0 — Engineering Foundation | `1b7980c81fb62a08c358f5ef5dc828cdcc9d32a2` | Go modular monolith, PostgreSQL/configuration/HTTP lifecycle, migration and CI foundations, initial Next.js health surface, ADRs 001–002. |
| 1 — Campaign Control Plane | `ee0e625f9bdab4589631422c401385790457e190` | Advertiser and campaign domain/API, transactional lifecycle/targeting rules, optimistic preconditions and idempotency, product frontend, ADRs 003–004. |
| 2 — Deterministic Ad Decision Engine | `6fcaf1d1ee011927d7a7a88e086050c5632136a3` | Direct-PostgreSQL eligibility/ranking, bounded diagnostics, Decision Lab, load fixture and [Phase 2 baseline](performance/phase2-baseline.md), ADRs 005–006. |
| 3 — Strict Budget Accounting Baseline | `fcdd7a4595e936d1340f9260ff77b39322a29bc4` | Serialized exact budget consumption, immutable receipts, financial idempotency, accounting verification, load fixture and [Phase 3 baseline](performance/phase3-baseline.md), ADRs 007–009. |

Roadmap V2 preserves these phases as the correctness baseline. Subsequent phases must record evidence through their ExecPlans and durable reports rather than rewriting this history.

