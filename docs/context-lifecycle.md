# Codex context lifecycle

## Central invariant

> After a context checkpoint, the entire Codex conversation can be discarded because repository state contains everything needed to resume safely.

Engineering rigor is never traded for fewer tokens. The optimization target is minimum waste per unit of verified engineering progress.

## New phase

Use a fresh conversation. Read, in order:

1. `AGENTS.md`
2. `docs/handoffs/latest.md`
3. `docs/phases/phase-XX.md`

Before substantial implementation, create `docs/plans/phase-XX-execplan.md`. Use `docs/context-map.md` for targeted expansion. Do not load every previous phase.

## Resume

Read `AGENTS.md`, the latest handoff, the active ExecPlan, the current phase spec, and then only routed context. Conversation history must not be required.

## Narrow-first inspection

Use this progression:

```text
search -> targeted range -> callers/references/tests -> full file only when required
```

Avoid recursive documentation/source dumps, every test or ADR, giant Git history, and giant tool output. Optimize relevant context per token, not minimum tool calls.

## ExecPlan contract

A substantial plan records:

```text
Status:
Current milestone:
Completed:
Verified:
Remaining:
Known failures/limitations:
Relevant files:
Relevant ADRs:
Next-session starting point:
```

It may include `## Decisions` and must include `## Evidence Index`:

| Check | Command / method | Result | Evidence |
| --- | --- | --- | --- |
| focused tests | exact command | actual result | concise note or log path |
| integration | exact method | actual result | evidence |
| failure test | exact scenario | actual result | report |
| broad gate | exact command | actual result | log |

The index prevents future sessions from reopening large historical logs.

## Verification and output

Stage work as focused check, focused integration/concurrency/failure verification, repair, focused confirmation, then one broad completion gate. Do not rerun the full project after every edit or skip the meaningful final gate.

For large successful commands, preserve full logs externally when useful and report command, exit status, counts, important warnings, and evidence path. For failures, retain the failure excerpt, inspect relevant code/tests, form a hypothesis, and run a focused check. Do not repeatedly tail unchanged output.

Never save tokens by weakening reasoning, skipping concurrency/failure/security work, mocking away important infrastructure, accepting uncertain correctness, hiding failures, or shrinking scope without authorization.

## Hard context checkpoint

Checkpoint at a coherent engineering boundary, not an arbitrary token threshold. Typical triggers are:

- implementation and focused tests are complete, with substantial integration/failure work remaining;
- concurrency/failure work is complete, with substantial repair/regression/documentation remaining;
- debugging history is large and the remaining work has a clean resumable boundary.

When triggered, finish only the current milestone, update the ExecPlan and Evidence Index, update the handoff if needed, leave a coherent repository, report the checkpoint, end with exactly `CONTEXT CHECKPOINT READY`, and stop. Do not continue substantial work or wait for automatic compaction.

## Two-agent workflow

The Builder owns implementation, focused tests, required migrations, focused integration, and implementation-caused documentation. A fresh Adversarial Reviewer reads the repository instructions, phase spec, ExecPlan/Evidence Index, relevant diff or commit range, and relevant invariants—not everything.

The reviewer seeks broken invariants, races, transaction mistakes, unsafe retries, duplicate effects, ordering/staleness bugs, unbounded cardinality, migration hazards, weak tests, missing failures, security/privacy issues, unsupported performance claims, and architecture without evidence. Builder and reviewer avoid duplicate work.

## Optional usage evidence

Future phases may record model responses, tool calls, token/cache counts, compactions, and session count when readily available. Do not spend meaningful context manufacturing unavailable metrics.

