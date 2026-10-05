# Phase 4 — Performance Baseline & Bottleneck Characterization

**Status: NOT STARTED**

## Goal

Produce a reproducible, honest performance characterization of Mercury's completed Phase 3 architecture and identify measured constraints, if any.

## Why it exists

The direct-PostgreSQL decision path and serialized budget accounting are simple, strong correctness baselines. Mercury must learn where they stop satisfying explicit objectives before introducing asynchronous propagation, projections, caches, messaging, or service boundaries.

## Prerequisites

- Explicit user authorization in a fresh Codex conversation.
- Read `AGENTS.md`, `docs/handoffs/latest.md`, this spec, and routed context.
- Create `docs/plans/phase-04-execplan.md` before substantial work.
- Preserve Phase 0–3 invariants and existing performance evidence.

## Questions

- Which current workloads and datasets are representative enough to reveal constraints?
- What throughput and p50/p95/p99 latency does each path achieve, with what error rate?
- Where are CPU, memory, allocations, GC, connection-pool waits, query time, locks, candidate cardinality, and database saturation spent?
- Does the direct-PostgreSQL path fail an explicit Mercury objective? At what environment, dataset, and concurrency?
- Which finding, if any, justifies a later architectural change?

## Non-goals

- Implementing reservations or changing budget semantics.
- Adding Kafka, Redis, a serving projection, or another datastore.
- Extracting services or deploying Kubernetes.
- Optimizing before the baseline identifies a bottleneck.
- Claiming Delivery Hero or internet-scale performance.
- Starting Phase 5.

## Allowed changes

- Reproducible fixtures, load workloads, benchmarks, profiling support, and bounded diagnostic instrumentation needed for measurement.
- Performance reports and documentation.
- Minimal frontend display of real collected evidence only when it improves comprehension.
- A narrowly justified measurement fix that does not change business semantics.

## Forbidden changes

- Changes to campaign, decision, or budget correctness contracts for performance.
- Unmeasured architecture additions or technology adoption.
- Fake metrics, unsupported extrapolation, or hidden errors.
- Production behavior changes unrelated to obtaining trustworthy measurements.

## Required measurements

For each reported scenario record:

- environment and relevant limits;
- dataset shape and campaign/candidate cardinality;
- concurrency and duration;
- RPS and p50/p95/p99;
- error count/rate and correctness checks;
- Go CPU, memory, allocations, and GC where relevant;
- database pool waits, query latency/plans, connections, and saturation evidence;
- workload and fixture version/command.

Reuse and compare `docs/performance/phase2-baseline.md` and `phase3-baseline.md` without presenting them as Phase 4 completion evidence.

## Verification

- Validate fixtures and semantic correctness before trusting load results.
- Use repeatable runs and explain material variance.
- Keep focused checks during instrumentation, then run the applicable broad completion gate once.
- If measurement behavior changes under multiple independent processes/connections, include the simplest meaningful multi-instance setup.
- Conduct an adversarial review for bad fixtures, coordinated omission, hidden errors, resource limits, unsupported conclusions, and accidental behavior changes.

## Evidence

The ExecPlan Evidence Index must link commands, actual results, concise environment metadata, retained logs/reports, query/profile artifacts, failures, and limitations. Label results `MEASURED`, conclusions `DERIVED`, and untested forecasts `HYPOTHETICAL`.

## Definition of done

- A peer can reproduce the baseline.
- Current constraints are identified with evidence—or the report explicitly shows no tested objective was exceeded.
- Architecture recommendations follow from measurements and include rejected alternatives/tradeoffs.
- Phase 0–3 correctness remains intact and the applicable completion gate passes.
- The handoff and roadmap reflect actual evidence without starting later work.

## Hard stop

Phase 4 characterizes the current architecture. It does not introduce future architecture. Stop after the completion report and wait for explicit authorization before Phase 5.

