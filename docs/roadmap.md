# Mercury Roadmap V2

## Governing question

> How does a simple, strongly correct advertising decision system evolve into a high-throughput, low-latency, horizontally scalable and failure-tolerant serving platform while keeping economic inconsistency explicit, measurable and bounded?

The progression is deliberate:

```text
simple and strongly correct
-> measure limitations
-> identify real bottlenecks
-> separate control/data concerns
-> propagate durable changes
-> build projected/local serving state
-> prove horizontal replication
-> relax consistency only where justified
-> bound business error
-> test overload, failure, and operations
```

Technology named in a future phase is not authorized before that phase. Each phase requires explicit user authorization and stops when its contract is complete.

## Cross-cutting completion gates

A phase is not complete merely because code exists. Select evidence appropriate to its risks: focused/unit tests, real integration, migrations, concurrency and multi-instance tests, failure/security tests, lint/static analysis, builds, runtime verification, performance evidence, documentation, adversarial review, and known limitations.

Any phase adding mutable shared state, distributed counters, caches/projections, background workers, schedulers, consumers, leases, caps, pacing, allocations, or serving replicas must ask whether multiple independent processes change correctness. If yes, single-process verification is insufficient. Use multiple processes, Compose replicas, independent database connections/consumers, or a local load-balancing boundary as appropriate. Kubernetes is not required for this proof.

Cross-instance correctness should use justified shared authority, constraints, transactions, idempotency, versions, durable streams, explicit partition ownership or leases, and bounded inconsistency. Do not prescribe distributed locking before defining the invariant and coordination need.

For APIs, event schemas, projection formats, database schemas, and configuration, consider N/N+1 coexistence. Prefer compatible evolution, expand/migrate/contract, versioned events, and tolerant readers where relevant; never assume an atomic fleet rollout.

Performance claims must identify environment, dataset, concurrency, duration, throughput, percentiles, error rate, resource use, and dependency saturation. Separate measured, derived, and hypothetical results.

## Frontend product strategy

Mercury retains one coherent, human-facing frontend whose purpose is to make domain state and distributed-system transformations understandable and operable without curl, database, or log inspection. It is never authoritative for legality, eligibility, ranking, budget, pacing, caps, projection truth, ownership, or failure policy; it submits commands and renders backend/runtime truth.

The **Vendor Advertising Console** may grow with real domain capabilities: campaign list/create/detail, lifecycle, targeting, budget, frequency cap, pacing, serving preview, and honest loading/error/empty/status states with responsive accessible interaction.

The **Operations / Engineering View** may expose real RPS/latency/errors, candidate counts, source and serving versions, propagation/freshness, cap behavior, pacing error, allocations and bounded overspend, lag, readiness/bootstrap, degradation/load shedding, regions, divergence, and SLOs. It is not a generic cluster dashboard and must never invent metrics.

Frontend capability follows backend truth: Phase 4 may expose performance evidence; 5 source/serving versions; 6 bootstrap/freshness; 7 serving latency/replicas; 8 cap behavior; 9 pacing; 10 allocation topology; 11 SLOs; 12 degradation; 15 readiness/rollout; 16 regional divergence.

React and TypeScript have public company evidence; Next.js remains a Mercury project choice. Any substantial frontend architecture change is deliberate and separately scoped. Microfrontend extraction requires an ADR and demonstrated independent ownership, deployment/release cadence, fault isolation, or runtime needs.

## Phase 0 — Engineering Foundation

**Status: COMPLETE**

Established the Go modular monolith, PostgreSQL lifecycle, configuration, health/readiness, graceful HTTP shutdown, migrations, CI and test boundaries, and initial Next.js operations shell. Historical behavior remains documented in `docs/architecture.md` and the original ADRs.

## Phase 1 — Campaign Control Plane

**Status: COMPLETE**

Implemented advertisers, campaign configuration/lifecycle/targeting, optimistic HTTP preconditions, transactional invariants, and idempotent creation. PostgreSQL is authoritative.

## Phase 2 — Deterministic Ad Decision Engine

**Status: COMPLETE**

Established the correctness-first direct-PostgreSQL decision path, exact eligibility and deterministic ranking, transient decision identity, bounded diagnostics, and reproducible baseline evidence. This baseline is intentionally retained for comparison.

## Phase 3 — Strict Budget Accounting Baseline

**Status: COMPLETE**

Established exact minor-unit money, campaign-row serialization, immediate consumption, immutable receipts, financial idempotency, and counter/ledger verification. It is the strong globally coordinated comparison point for later budget work.

## Phase 4 — Performance Baseline & Bottleneck Characterization

**Status: NOT STARTED**

**Question:** Where does the current direct-PostgreSQL architecture actually stop satisfying Mercury's serving objectives?

Measure representative campaign datasets and concurrency: RPS, p50/p95/p99, errors, Go CPU/memory/allocations/GC, database pool waits, query latency/plans/saturation, and candidate counts. Tools may include k6, pprof, Go benchmarks, and PostgreSQL execution evidence.

The output is a reproducible baseline and measured bottleneck(s), if any. It is valid for PostgreSQL to perform better than expected. Do not add Redis, Kafka, extract services, or optimize without evidence. Frontend work is allowed only when useful to display real evidence. See `docs/phases/phase-04.md`.

## Phase 5 — Campaign Change Propagation Backbone

**Question:** How does authoritative campaign state propagate durably to high-volume serving infrastructure?

Define durable publication, campaign versions, ordering, duplicate delivery, replay, consumer lag, schema evolution, and freshness semantics. Kafka requires a justified ADR; naming it here is not a decision to adopt it. UI may expose real source and serving versions.

## Phase 6 — Serving-State Projection & Bootstrap

**Question:** How can serving avoid synchronous control-plane database dependency while remaining boundedly fresh and rebuildable?

Explore projections, in-memory indexes, snapshot/bootstrap, event catch-up, versions, staleness, rebuild, and new-replica startup. Redis is not automatic; compare local memory and shared fast state from evidence.

### Multi-instance requirement

Every serving replica must independently start, obtain a baseline, catch up durable changes, determine its projection version, detect unacceptable staleness, and recover after restart. Design and verify new replicas starting during updates, snapshot/event race windows, duplicate application, out-of-order changes, rebuild, and restarts with empty or stale state. No replica may require a particular sibling to remain alive.

### Readiness

Being alive is insufficient for serving readiness. A conceptual lifecycle is `STARTING -> BOOTSTRAPPING -> CATCHING_UP -> READY`; the exact state machine is a later design decision. Readiness requires explicit bootstrap/freshness criteria. UI may expose real state, version, freshness, and readiness.

## Phase 7 — Low-Latency Serving Data Plane

**Question:** Can Mercury meet an explicit serving latency SLO using projected state?

Focus on candidate retrieval, targeting, eligibility, ranking, locality, the hot path, and availability. Set an SLO from measured Mercury behavior, not company claims.

### Replicated-serving verification

Run multiple independent serving processes before Kubernetes. Distribute concurrent requests—including requests for the same campaign, viewer, budget, and targeting state—across different replicas. Verify independence, consistent interpretation, safe bootstrap/restart, traffic redistribution, stale-replica handling, and readiness. Correctness cannot depend on affinity unless ownership/affinity is intentionally designed and documented. UI may expose real serving latency and replica diagnostics.

## Phase 8 — Frequency Capping & Distributed Eligibility State

**Question:** How do multiple serving replicas enforce viewer/campaign exposure limits without globally synchronous coordination on every request?

Compare strict global/central atomic state, regional state, local approximation, bounded violation, TTL/window semantics, high-cardinality behavior, and privacy. Test the explicit race where replicas A, B, and C each observe count 2 and receive concurrent requests for the same synthetic viewer/campaign. Declare the guarantee as `STRICT`, `BOUNDED`, `REGIONAL`, or `APPROXIMATE` and prove it across replicas; never hide the race behind one-process tests. UI exposes real configuration and enforcement behavior.

## Phase 9 — Deterministic Budget Pacing

**Question:** How should spend be distributed across campaign lifetime?

Model expected and actual spend, pacing error, remaining budget/time, and available traffic. Test uniform traffic, lunch spikes, surges, collapse, and undersupply; do not introduce ML.

Define who computes and owns pacing state, how replicas observe it, permitted staleness, and whether it is centralized, projected, partitioned, or derived. Test that multiple replicas acting on it do not create unacceptable global behavior. Never assume a singleton pacing loop without explicit ownership. UI shows backend-derived expected/actual spend and pacing state.

## Phase 10 — Distributed Budget Allocation & Bounded Overspend

**Question:** Which coordination/consistency model improves serving scalability while keeping economic error explicit and bounded?

Use Phase 3's global PostgreSQL serialization as the strong baseline. Compare conditional database mutation, central authority, reservations, tokens, leases, regional allocation, and bounded overspend plus narrow reconciliation.

Define the relaxed guarantee, allowed error/overspend, allocation ownership, shared versus local state, replenishment, lease semantics if any, crash behavior, unused allocation recovery, and double-consumption prevention. Concurrently test replicas 1, 2, and 3 consuming allocations; no correctness argument may assume one serving process. UI may visualize real global budget, allocations, consumption, reserve/reallocation, and bounded error.

## Phase 11 — Observability & SLO Engineering

Adopt only a justified, target-compatible stack; candidates include OpenTelemetry, Prometheus, Grafana, and Tempo. Define SLOs for serving latency/availability, campaign propagation, projection freshness, cap error, pacing, and budget error. Control cardinality and operational cost. Instrumentation must answer real questions, and the UI may expose truthful SLO state.

## Phase 12 — Overload, Degradation & Failure Engineering

Study timeouts, retry amplification, backpressure, load shedding, admission control, dependency isolation, and circuit breaking only where justified.

Exercise PostgreSQL slow/unavailable; messaging lag/unavailable if present; projection/cache failure; consumer lag; traffic surge; response loss; and stale campaign state. Instance-level experiments include killing one and several serving replicas under load, restarting replicas with empty and stale projections, starting a replica during heavy propagation, uneven load, and killing/restarting a background consumer or ownership holder.

For each experiment record what failed and stayed correct, business impact and request errors, traffic redistribution/takeover, dead-process invariant dependencies, stale bootstrap and early-readiness behavior, replay/duplicate effects, p95/p99, retry amplification, and recovery time. Findings inform Phase 15 rather than being deferred to Kubernetes. Write incident-style evidence for meaningful results. UI may expose real degraded modes.

## Phase 13 — High-Scale Go Performance Engineering

Optimize measured bottlenecks only. Use pprof, benchmarks, trace, runtime metrics, GC/allocation/CPU profiles, lock/scheduler analysis, connection pressure, and serialization cost as relevant. Every optimization records before, change, after, and tradeoff; intuition-only performance claims are invalid.

## Phase 14 — Service Extraction Where Justified

Evaluate extraction only from evidence about independent scaling, availability, fault isolation, deployment, ownership, data lifecycle, or latency. Candidate boundaries include control plane, serving data plane, and projector/propagation; keeping fewer deployables remains valid.

Every extraction requires an ADR and a replication model: whether it can run N replicas; local versus authoritative state; partition ownership; whether a true singleton responsibility justifies leader election; instance-death behavior; replacement state reconstruction; and mixed-version operation. A microservice is not automatically stateless. Do not introduce leader election without a real singleton responsibility.

## Phase 15 — Kubernetes + Cloud Productionization

**Central question:** Now that Mercury's components are already designed and tested for horizontal replication, how should they be deployed, replaced, scaled, drained, and recovered in a production-style environment?

Kubernetes is an operations phase, not an application-correctness mechanism. Kubernetes does not make Mercury multi-instance safe.

### Preconditions

Components intended for replication already need evidence for multi-instance correctness, instance-independent request handling, safe restart, reconstructable local state, correct readiness semantics, graceful shutdown, and bounded bootstrap. If these are absent, fix application architecture before continuing Kubernetes work.

### Target infrastructure

Use only justified infrastructure. Docker, Kubernetes, Helm, Terraform, and a cloud provider are candidates, not résumé requirements. GCP remains preferred only while current public Delivery Hero AdTech evidence and Mercury's needs support it; otherwise classify the provider as a project choice. Never infer proprietary topology.

### Replica management

Demonstrate multiple serving replicas, traffic distribution, pod replacement, safe state reconstruction, and replica independence.

### Liveness and readiness

Liveness asks whether the process functions. Readiness asks whether this instance is safe for its intended traffic. A process may be alive but unready while bootstrapping, catching up, recovering state, or draining. Do not route before explicit readiness criteria pass.

### Graceful termination

Withdraw readiness, stop new routing, boundedly complete/terminate in-flight work according to contract, close consumers/resources and release ownership safely, then exit before the deadline. Test shutdown under load; do not claim zero loss without workflow-specific evidence.

### Rolling deployment

Verify coexistence of `vN`, `vN`, and `vN+1`, including API, event, projection, database, and configuration compatibility plus rollback where practical. Do not assume one version at a time.

### Horizontal autoscaling

Choose a measured signal such as CPU, concurrency, RPS, backlog, or latency. Measure scale-out delay, new-replica bootstrap/readiness delay, latency during scaling, capacity gained, and scale-in behavior. HPA is not added for appearance.

### Resource management

Define and test CPU/memory requests and justified limits. Observe CPU/memory pressure, OOM/restart, and resource scarcity.

### Failure recovery

Delete one and several serving pods under load, restart projection consumers, and simulate node loss where practical. Measure error rate, p95/p99, recovery time, bootstrap duration, and capacity impact.

### Service discovery and routing

Use Kubernetes networking for actual needs. Do not automatically add a service mesh, custom operator, sidecar architecture, or complex ingress.

### Observability

Correlate orchestration behavior with serving, freshness, pacing, budget, and availability SLOs. “Are pods green?” is not the success criterion.

### Required evidence

Record multi-replica steady state, pod loss under load, replacement/bootstrap, readiness gating, graceful shutdown, rolling deployment/rollback, autoscaling, and resource pressure. Include RPS, p50/p95/p99, errors, replica count, bootstrap/readiness/recovery times, CPU, and memory.

### Non-goals

Do not add a service mesh, custom operators, multiple clusters, GitOps platform, complex ingress stack, or cross-region orchestration solely for sophistication.

### Definition of done

Kubernetes operates Mercury's already-correct replicated architecture through deployment, replacement, scaling, rollout, and recovery without hidden singleton dependencies or violated correctness/availability contracts. UI may expose useful real readiness/rollout state, not become a cluster-management product.

## Phase 16 — Multi-Region Serving

**Central question:** How does Mercury serve across regions when state and economic authority cannot remain perfectly synchronous?

This phase is not “run Kubernetes in two regions.” It is primarily a distributed-systems semantics problem: campaign and budget ownership, propagation/replication latency, network partitions, regional budget allocation, regional frequency-cap semantics, regional serving independence and traffic, outage/failover, recovery/rejoin, and bounded divergence.

Study partition behavior and what each region may safely do, then prove failure and rejoin semantics. Kubernetes may host workloads, but infrastructure is not the answer to cross-region consistency. UI may expose real regional health and divergence.

## Phase 17 — Capacity Planning & Production Economics

Build a capacity/cost model from measured RPS per instance, CPU, memory, network, event throughput/partitions/storage, projection/cache memory, replication, observability cost, cloud cost, and headroom. Ask what 10x tested load requires and whether the architecture is economically reasonable. Label every input or conclusion `MEASURED`, `DERIVED`, or `HYPOTHETICAL`.

## Phase 18 — Final Adversarial Audit, Case Study & Oral Defense

Introduce no large architecture. Audit correctness, replica safety, failure, security/privacy, performance/capacity, Kubernetes operations, multi-region semantics, documentation consistency, company evidence, and portfolio boundary.

Produce a case study covering the initial architecture, measurements, bottlenecks, transitions, tradeoffs/rejected alternatives, consistency relaxation and bounded error, failure behavior, performance, operations, cost, and limitations. Prepare oral-defense questions. Completion requires repository evidence sufficient for senior/staff-level scrutiny.

