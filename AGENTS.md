# Mercury repository instructions

## Context discipline

Engineering quality outranks token reduction. Repository state is durable project memory; a fresh session must be able to resume without conversation history.

- Route reads through `docs/context-map.md`.
- For a new phase read this file, `docs/handoffs/latest.md`, and the current phase spec, then create the phase ExecPlan before substantial work.
- To resume a phase, also read its active ExecPlan.
- Search first, read targeted ranges, expand only when necessary, reuse unchanged understanding, and prefer diffs over rereading whole files.
- Do not routinely load the archived master prompt, every ADR or document, previous phase specs, full Git history, progress history, or all performance reports.

## Phase discipline

- Work only on one explicitly authorized phase. Never start the next phase automatically.
- A technology appearing in the roadmap does not authorize introducing it early.
- Earn complexity with measured evidence. Preserve completed Phase 0–3 semantics unless an authorized phase explicitly changes them.

## Engineering discipline

- PostgreSQL is the current authoritative transactional source.
- Use exact money representations and explicit time semantics.
- Never hide races, partial failure, stale state, duplicate delivery, ordering problems, retry ambiguity, or bounded inconsistency.
- Serving optimization is measurement-driven. Never fabricate scale or performance claims.
- Extract services only when evidence about scaling, availability, ownership, deployment, fault isolation, or lifecycle warrants it.
- For APIs, events, projections, database schemas, and configuration, consider mixed-version N/N+1 operation; do not assume atomic fleet deployment.

## Multi-instance invariant

Assume every independently deployable backend component may eventually run concurrently as multiple instances. Business correctness must not accidentally depend on a singleton process, local locks/counters/durable state, sticky sessions, one scheduler or consumer, or one process seeing every request for an entity. Any such dependency must be explicitly designed, bounded, documented, and verified. Process-local state must be instance-scoped or derived/reconstructable, non-authoritative across instances, and safe to lose.

Kubernetes does not make Mercury multi-instance safe. Prove replicated application semantics first; orchestration comes later. Do not default to distributed locks—define the invariant and actual coordination need first.

Any phase whose correctness may change with replicated mutable state, workers, schedulers, consumers, projections, caps, pacing, allocations, or serving replicas must answer whether multiple processes alter semantics. If yes, single-process verification is insufficient; use the simplest suitable multi-process setup before Kubernetes.

## Verification

Use focused checks while implementing, real integration where dependency behavior matters, and concurrency, failure, and security tests where relevant. Finish with one broad completion gate. Do not repeatedly rerun unchanged successful suites, weaken tests, mock away material behavior, or fabricate evidence. Complete phases with an adversarial review and documented limitations.

## Context checkpoint

Follow `docs/context-lifecycle.md`. A triggered checkpoint is terminal for the conversation: update the ExecPlan and Evidence Index, leave a coherent state, and end the final response with exactly:

```text
CONTEXT CHECKPOINT READY
```

