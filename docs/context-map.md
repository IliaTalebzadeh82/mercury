# Context routing map

## Startup

| Situation | Required context |
| --- | --- |
| New phase | `AGENTS.md` + `docs/handoffs/latest.md` + current phase spec |
| Resume phase | above + active `docs/plans/phase-XX-execplan.md` |

Then search and expand only through the relevant route.

| Work area | Required context |
| --- | --- |
| Campaign control | `docs/domain-model.md` + `docs/invariants.md` + relevant campaign/advertiser code and tests + ADRs 003/004 |
| Decision serving | domain/invariants + `internal/decision` and API tests + ADRs 005/006 |
| Budget | domain/invariants + `internal/budget` and tests + ADRs 007–009 |
| Performance | current phase spec + relevant `docs/performance/` report + load test + measured hot-path code |
| Campaign propagation | roadmap Phase 5 + architecture/invariants + future propagation ADR/code/tests |
| Serving projections/bootstrap | roadmap Phase 6 + architecture/invariants + consistency/failure evidence |
| Multi-instance correctness | invariants + architecture + current phase + relevant source/tests |
| Frequency caps | roadmap Phase 8 + privacy/state semantics + relevant tests/evidence |
| Pacing | roadmap Phase 9 + budget/domain invariants + traffic evidence |
| Distributed budget | roadmap Phase 10 + budget ADRs/invariants + concurrency/failure evidence |
| Failure engineering | roadmap Phase 12 + architecture + relevant SLOs, incident evidence, and tests |
| Kubernetes | roadmap Phase 15 + proven multi-instance evidence + readiness/shutdown/mixed-version contracts |
| Multi-region | roadmap Phase 16 + consistency/ownership contracts + budget/propagation evidence |
| Frontend | roadmap phase slice + frontend boundary in roadmap + `web/` feature/tests + backend contract |

## Never default-load

- `docs/archive/masterprompt-original.md`
- all historical phase specs
- every ADR
- all Git history
- `docs/progress.md`
- every performance report

These remain discoverable evidence, not routine startup context.

