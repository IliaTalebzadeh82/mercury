# Mercury portfolio boundary

Mercury asks:

> How does a simple, strongly correct advertising decision system evolve into a high-throughput, low-latency, horizontally scalable and failure-tolerant serving platform while keeping economic inconsistency explicit, measurable and bounded?

## Deep ownership

Mercury deeply owns real-time serving; candidate retrieval, targeting, ranking, and eligibility; campaign propagation and serving projections; low-latency Go performance; frequency capping; budget pacing and distributed allocation; bounded inconsistency; replica safety; failure isolation, overload, degradation, and SLOs; Kubernetes operation; multi-region serving; and capacity/economic analysis.

## Minimal supporting ownership

Mercury may implement only the serving/impression events and downstream semantics needed to support the serving problem. Reconciliation stays narrow: authoritative campaign state versus serving projection, and allocated budget versus consumed budget.

## Deliberate exclusions

- Financial-network reconciliation and payment settlement belong in a money-movement project.
- Deep streaming analytics/OLAP belongs in a high-volume analytics project.
- A full multi-touch attribution platform is not core scope.
- A generic durable workflow engine belongs elsewhere.
- ML-heavy ranking is optional research; deterministic ranking remains sufficient unless deliberately revisited.
- Automated bidding optimization is not required for project completion.

These boundaries prevent supporting systems from displacing Mercury's central distributed-serving problem.

