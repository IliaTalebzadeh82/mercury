# Original master-prompt migration review

The exact original prompt is preserved at `docs/archive/masterprompt-original.md`. It is audit material and is never default startup context. This review indexes every meaningful original section and records its disposition under Roadmap V2.

Disposition meanings: `PRESERVED`, `UPDATED`, `MOVED`, `NARROWED`, `DE-SCOPED`, `OPTIONAL`, and `SUPERSEDED`.

## Product and engineering foundations

| Original section | Disposition | Durable destination / rationale |
| --- | --- | --- |
| 1 Primary Product Question | `UPDATED` | `docs/portfolio-boundary.md` and `docs/roadmap.md` use the new evolution/explicit-bounded-inconsistency question. |
| 2 Core Engineering Philosophy | `PRESERVED` | `AGENTS.md`, roadmap progression, and phase/evidence gates preserve correctness-first evolution. |
| 3 Earn Complexity | `PRESERVED` | `AGENTS.md`, Phase 4, and roadmap phase discipline require measured need. |
| 4 Technology Contract | `UPDATED` | Current choices remain historical facts; future choices are phase- and evidence-gated. |
| 4.1 Backend Language | `PRESERVED` | Go remains the backend language; public alignment is in `company-evidence.md`. |
| 4.2 HTTP Stack | `PRESERVED` | Current standard-library HTTP boundary stays in architecture/code map; future change needs evidence. |
| 4.3 Transactional Database | `PRESERVED` | PostgreSQL remains current authority in `AGENTS.md`, architecture, and ADR 002. |
| 4.4 Database Access | `PRESERVED` | Existing explicit `pgx` approach remains historical/current code; no migration change. |
| 4.5 Database Migrations | `PRESERVED` | Existing migration discipline remains in README/architecture and phase completion evidence. |
| 4.6 Configuration | `PRESERVED` | Current typed startup validation remains; mixed-version configuration is now explicit. |
| 4.7 Logging | `PRESERVED` | Existing structured logging remains; observability evolves in Phase 11. |
| 4.8 Validation | `PRESERVED` | Domain/API validation remains in current code and invariants. |
| 4.9 Testing — Backend | `UPDATED` | `AGENTS.md`, context lifecycle, and completion gates add staged, multi-instance and adversarial verification. |
| 4.10 Messaging | `UPDATED` | Messaging is no longer presumed; Phase 5 may justify Kafka through an ADR. |
| 4.11 Event Format | `MOVED` | Phase 5 owns schema/version/ordering design; mixed-version rules are cross-cutting. |
| 4.12 Cache / Fast State | `UPDATED` | Phase 6 compares local/shared projected state from evidence; Redis is not automatic. |
| 4.13 Analytical Database | `DE-SCOPED` | Deep OLAP is outside Mercury's portfolio boundary. |
| 4.14 Observability | `MOVED` | Phase 11 owns justified instrumentation and SLOs. |
| 4.15 Profiling | `MOVED` | Phase 4 characterizes; Phase 13 optimizes measured Go bottlenecks. |
| 4.16 Load Testing | `PRESERVED` | Phase 4 spec and performance-claim discipline require reproducible workload evidence. |
| 4.17 Local Infrastructure | `UPDATED` | Use the simplest phase-appropriate environment, including local multi-process verification before Kubernetes. |
| 4.18 CI | `PRESERVED` | Current CI remains; future gates follow phase evidence needs. |
| 4.19 Deployment | `MOVED` | Phase 15 owns Kubernetes productionization after replication correctness. |
| 4.20 Cloud | `UPDATED` | Phase 15 selects a provider from project need plus revalidated public evidence. |

## Frontend and repository design

| Original section | Disposition | Durable destination / rationale |
| --- | --- | --- |
| 5 Frontend Technology Contract | `UPDATED` | Roadmap frontend strategy preserves the UI while requiring deliberate architecture decisions. |
| 5.1 Frontend Framework | `UPDATED` | React/TypeScript are publicly evidenced; Next.js is explicitly `PROJECT_CHOICE`. |
| 5.2 Package Manager | `PRESERVED` | Current pnpm contract remains in repository/README; not promoted to global architecture. |
| 5.3 UI System | `NARROWED` | Existing components remain; future UI must be coherent, accessible, and domain-driven, not stack-driven. |
| 5.4 Client-Side Data Fetching | `PRESERVED` | Current server boundary remains; frontend renders backend truth. |
| 5.5 Forms | `PRESERVED` | Forms remain part of the Vendor Console when backed by domain commands. |
| 5.6 Tables | `PRESERVED` | Tables remain a presentation tool where domain workflows need them. |
| 5.7 Charts | `NARROWED` | Charts are allowed only for real backend/runtime evidence; no fake dashboards. |
| 5.8 Frontend Testing | `PRESERVED` | Unit/integration/E2E expectations remain in verification and public-evidence policy. |
| 5.9 Frontend State Policy | `UPDATED` | Frontend is explicitly non-authoritative and may display intermediate distributed state. |
| 5.10 Frontend Responsibility Boundary | `PRESERVED` | Roadmap lists all rules the frontend cannot own. |
| 6 Client Experiences | `UPDATED` | One coherent product contains Vendor Advertising Console and Operations/Engineering View. |
| 6.1 Advertiser Console | `UPDATED` | Renamed Vendor Advertising Console and bounded to meaningful campaign workflows. |
| 6.2 Mercury Operations Console | `UPDATED` | Operations View follows real phase/runtime truth and is not a Kubernetes dashboard. |
| 7 Frontend Design Principles | `PRESERVED` | Human-friendly, responsive, accessible loading/error/empty/status semantics remain required. |
| 8 Monorepo Structure | `UPDATED` | Existing code remains; `code-map.md` and context structure add durable navigation without speculative packages. |
| 9 Package Design — Go | `PRESERVED` | Modular ownership remains in `code-map.md`; extraction requires Phase 14 evidence/ADR. |
| 10 Domain Language | `PRESERVED` | Existing domain model remains; future phase specs define new terms before implementation. |

## Correctness, testing, and operations

| Original section | Disposition | Durable destination / rationale |
| --- | --- | --- |
| 11 Core Domain Invariants | `PRESERVED` | Completed Phase 1–3 invariants remain in `docs/invariants.md`; future rules are additive. |
| 11 Campaign lifecycle | `PRESERVED` | Phase 1 transition/editability rules remain in `docs/invariants.md`. |
| 11 Budget | `UPDATED` | Phase 3 strict accounting remains the baseline; Phase 10 may explicitly relax coordination with bounded error. |
| 11 Reservation lifecycle | `NARROWED` | Not an automatic next feature; reservation/token designs may be evaluated in Phase 10 if evidence warrants. |
| 11 Impression accounting | `NARROWED` | Only minimal serving/impression events needed by core serving phases remain in scope. |
| 11 Click accounting | `DE-SCOPED` | Not deep core scope; may exist only as minimal downstream semantics if required. |
| 11 Conversion attribution | `DE-SCOPED` | Full attribution is explicitly outside the portfolio boundary. |
| 11 Accounting | `UPDATED` | Strong Phase 3 accounting is preserved; later allocation reconciliation is deliberately narrow. |
| 11 Campaign propagation | `MOVED` | Phase 5 owns versioned, durable, replayable propagation and freshness. |
| 12 Money Rules | `PRESERVED` | Exact representation and explicit economic error remain global rules. |
| 13 Time Rules | `PRESERVED` | Explicit time semantics remain in `AGENTS.md`; later window/pacing phases define details. |
| 14 API Design | `PRESERVED` | Existing API contracts remain; N/N+1 compatibility is now cross-cutting. |
| 15 Idempotency | `PRESERVED` | Existing financial/control semantics remain; future retry/duplicate behavior must be explicit. |
| 16 Transactions | `PRESERVED` | Current transactional baseline remains and future relaxations require evidence. |
| 17 Do Not Hide Distributed-Systems Problems | `UPDATED` | `AGENTS.md`, invariants, and roadmap explicitly cover multi-instance races, stale state, ordering, retries, and bounded inconsistency. |
| 18 Testing Philosophy | `UPDATED` | Context lifecycle stages focused, integration, concurrency/failure, and final broad verification. |
| 18 Unit tests | `PRESERVED` | Focused checks remain the first verification layer. |
| 18 Integration tests | `PRESERVED` | Real dependency behavior is required where it matters. |
| 18 Concurrency tests | `UPDATED` | Global multi-instance gates now prevent one-process evidence from hiding replica races. |
| 18 Property/invariant tests | `PRESERVED` | Use where meaningful to validate durable invariants. |
| 18 Failure tests | `UPDATED` | Phase 12 adds explicit instance lifecycle, bootstrap, redistribution, and recovery scenarios. |
| 19 Performance Claims | `PRESERVED` | Roadmap and Phase 4 require environment/dataset/concurrency/percentile/resource evidence. |
| 20 Observability Philosophy | `PRESERVED` | Phase 11 links instrumentation to real questions and cost/cardinality. |
| 21 Metrics | `UPDATED` | Metrics follow phase SLOs and actual runtime truth; static catalog is not prebuilt. |
| 22 Distributed Tracing | `OPTIONAL` | Phase 11 may adopt tracing when it answers a demonstrated question. |
| 23 Documentation | `UPDATED` | Split into roadmap, context routing/lifecycle, phase specs, ExecPlans, evidence, handoff, and archive. |
| 24 ADR Policy | `PRESERVED` | Durable choices such as messaging, extraction, and microfrontends require ADRs. |
| 25 Event Design | `MOVED` | Phase 5 owns versioning, ordering, duplicates, replay, and lag; Phase 15 verifies mixed versions. |
| 26 Budget Accounting | `UPDATED` | Phase 3 is strong baseline; Phase 10 evaluates allocation/bounded overspend. |
| 27 Pacing | `MOVED` | Phase 9 owns deterministic pacing and replicated ownership/state propagation. |
| 28 Serving Path | `UPDATED` | Phases 4, 6, and 7 measure, project, and then prove low-latency replicated serving. |
| 29 Control Plane vs Data Plane | `MOVED` | Roadmap phases 5–7 introduce separation only after measurement. |
| 30 Attribution | `DE-SCOPED` | Full attribution is outside deep scope; only minimal serving events are allowed. |
| 31 Analytics | `DE-SCOPED` | Deep streaming analytics/OLAP is reserved for another project. |
| 32 Degradation Policies | `MOVED` | Phase 12 owns measured overload/degradation/failure behavior. |
| 32 PostgreSQL unavailable | `MOVED` | Current health/readiness behavior remains accurate; Phase 12 exercises future degradation. |
| 32 Kafka unavailable | `UPDATED` | Test only if Kafka is actually adopted in Phase 5. |
| 32 ClickHouse unavailable | `DE-SCOPED` | ClickHouse is not a required Mercury dependency. |
| 32 Campaign propagation delayed | `MOVED` | Phases 5–6 define freshness/staleness; Phase 12 injects delay. |
| 32 Budget subsystem degraded | `MOVED` | Phases 10–12 define guarantees and failure behavior. |
| 33 Reconciliation | `NARROWED` | Limited to source/projection and allocated/consumed budget comparisons. |
| 34 Failure Injection | `UPDATED` | Phase 12 adds replica death/restart/bootstrap/readiness/redistribution evidence. |
| 35 Multi-Region | `UPDATED` | Phase 16 is explicitly ownership/latency/partition/divergence, not multiple Kubernetes clusters. |
| 36 AI / ML Policy | `NARROWED` | Deterministic ranking is sufficient; ML-heavy ranking is optional and automated bidding is de-scoped. |
| 37 Security | `PRESERVED` | Security/privacy remains part of phase completion and final adversarial audit. |
| 38 Production Economics | `MOVED` | Phase 17 builds evidence-based capacity and cost models. |
| 39 Source Control Discipline | `PRESERVED` | Phase-scoped coherent changes remain; this migration does not commit or push. |
| 40 Strict Review Mode | `UPDATED` | Adversarial reviewer workflow and final phase audits make review resumable and targeted. |
| 41 Challenge Me | `PRESERVED` | Reviewer is explicitly tasked to challenge unsupported correctness/architecture claims. |
| 42 Phase Gating | `PRESERVED` | `AGENTS.md`, phase specs, handoff, and roadmap require explicit authorization. |
| 43 Phase Completion Report | `UPDATED` | ExecPlan Evidence Index plus phase-specific completion report replaces a giant prompt template. |

## Old roadmap and closing requirements

| Original section | Disposition | Durable destination / rationale |
| --- | --- | --- |
| 44 Roadmap overview | `SUPERSEDED` | `docs/roadmap.md` is canonical while preserving completed Phase 0–3 meaning. |
| Old Phase 0 | `PRESERVED` | Roadmap Phase 0 and `docs/progress.md`. |
| Old Phase 1 | `PRESERVED` | Roadmap Phase 1 and progress history. |
| Old Phase 2 | `PRESERVED` | Roadmap Phase 2 and Phase 2 performance evidence. |
| Old Phase 3 | `PRESERVED` | Roadmap Phase 3 and Phase 3 performance evidence. |
| Old Phase 4 Reservation Lifecycle | `SUPERSEDED` | Reservations are no longer an automatic next phase; they are one Phase 10 allocation alternative if evidence warrants. |
| Old Phase 5 Pacing | `MOVED` | Roadmap Phase 9, after propagation/projected serving/frequency caps. |
| Old Phase 6 Serving Data Plane | `UPDATED` | Split into Phase 6 projection/bootstrap and Phase 7 serving data plane. |
| Old Phase 7 Kafka Event Backbone | `UPDATED` | Roadmap Phase 5 defines a propagation problem; Kafka is conditional. |
| Old Phase 8 Measurement & Attribution | `DE-SCOPED` | Minimal serving events only; full attribution is outside core scope. |
| Old Phase 9 Analytical Platform | `DE-SCOPED` | Deep OLAP/analytics belongs in another portfolio project. |
| Old Phase 10 Observability & SLOs | `MOVED` | Roadmap Phase 11. |
| Old Phase 11 Failure Engineering | `UPDATED` | Roadmap Phase 12 includes replica lifecycle and overload evidence. |
| Old Phase 12 Reconciliation Platform | `NARROWED` | Narrow source/projection and budget allocation/consumption checks are embedded where needed. |
| Old Phase 13 High-Scale Serving | `UPDATED` | Roadmap Phase 13 is evidence-led Go performance; serving semantics are established earlier. |
| Old Phase 14 Multi-Region | `MOVED` | Roadmap Phase 16 after Kubernetes operations. |
| Old Phase 15 Advanced Ranking | `OPTIONAL` | Deterministic ranking remains sufficient unless later evidence and authorization justify research. |
| Old Phase 16 Automated Bidding | `DE-SCOPED` | Removed from required completion scope. |
| Old Phase 17 Productionization | `UPDATED` | Split into Phase 14 extraction, Phase 15 Kubernetes, Phase 17 economics, and Phase 18 audit. |
| 45 Prohibited Premature Choices | `PRESERVED` | Phase discipline and each roadmap non-goal prohibit premature technology. |
| 46 README Positioning | `UPDATED` | README links canonical roadmap, portfolio/evidence, context system, and archive. |
| 47 Portfolio / Resume Discipline | `UPDATED` | `portfolio-boundary.md` and evidence discipline reject breadth and unsupported scale claims. |
| 48 Final Project Character | `UPDATED` | Roadmap Phase 18 defines the evidence-based case study and oral defense. |
| 49 First Task | `SUPERSEDED` | Phases 0–3 are complete; `handoffs/latest.md` identifies unauthorized Phase 4 as next. |

## Completeness conclusion

No old requirement is silently deleted. Completed behavior and historical decisions remain intact; broad speculative requirements are either routed to a later evidence-gated phase, narrowed to support Mercury's core serving question, explicitly made optional, or deliberately de-scoped in the portfolio boundary.
