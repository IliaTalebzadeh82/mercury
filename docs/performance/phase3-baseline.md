# Phase 3 exploratory budget-accounting baseline

Measured 2026-09-26. These are single-development-machine observations, not
SLOs or production-capacity claims.

## Environment and method

- Host: Intel Core i7-10750H, 6 cores/12 threads, 15 GiB RAM, Linux x86-64.
- PostgreSQL: 17.6 Alpine in Docker Compose without configured CPU/memory limits.
- Backend: host process; PostgreSQL pool fixed at four connections.
- Driver: `grafana/k6:1.5.0` over HTTP.
- Fixture: one hot campaign, 100 independent campaigns, one replay campaign,
  one fully spent campaign, and the Phase 2 1,000-eligible-candidate cohort.
- Fixed-work consumption runs used 2,000 requests at concurrency 1/4/16/64.
- Mixed runs used 16 accounting VUs and 16 decision VUs for 15 seconds.
- All reported runs had zero HTTP failures and zero failed status checks.
- The accounting verification query returned zero discrepancy rows after the
  workload. PostgreSQL reported five sessions: four application pool
  connections plus the verification session.

## One hot campaign versus 100 campaigns

All requests used unique keys, amount one, and had sufficient budget.

| Distribution | Concurrency | Requests | Achieved req/s | p50 ms | p95 ms | p99 ms |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| One hot campaign | 1 | 2,000 | 812.78 | 1 | 2 | 2 |
| 100 campaigns | 1 | 2,000 | 793.55 | 1 | 2 | 2 |
| One hot campaign | 4 | 2,000 | 1,011.54 | 3 | 9 | 14.01 |
| 100 campaigns | 4 | 2,000 | 2,620.97 | 1 | 2 | 2 |
| One hot campaign | 16 | 2,000 | 1,020.68 | 16 | 17 | 17 |
| 100 campaigns | 16 | 2,000 | 2,568.19 | 6 | 8 | 13 |
| One hot campaign | 64 | 2,000 | 1,025.69 | 62 | 65 | 69 |
| 100 campaigns | 64 | 2,000 | 1,673.86 | 34 | 56 | 59 |

The hot campaign plateaus near 1,020 requests/second as its row becomes the
serialization point. Independent campaigns exploit the four-connection pool
and reach about 2.5 times that throughput at concurrency 4 and 16. At
concurrency 64, pool queuing raises latency for both; no pool or lock tuning was
performed.

## Direct lock and transaction timing

A separate real-PostgreSQL integration timing run issued 2,000 concurrent
commands through a pool fixed at four and inspected the durations returned by
the accounting transaction itself:

| Distribution | Lock p50/p95/p99 ms | Transaction p50/p95/p99 ms |
| --- | --- | --- |
| One hot campaign | 2.618 / 2.891 / 3.308 | 3.481 / 3.819 / 4.474 |
| 100 campaigns | 0.100 / 0.135 / 0.708 | 1.217 / 1.404 / 1.969 |

Transaction timing begins after pool acquisition. The HTTP measurements above
include pool waiting, JSON, routing, and network overhead.

## Replays and insufficient results

At concurrency 16:

| Workload | Requests | Achieved req/s | p50 ms | p95 ms | p99 ms | Durable effects |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| Same-key replay | 2,000 | 4,235.90 | 4 | 4 | 5 | 1 approval, 1,999 replays |
| Fully spent campaign | 2,000 | 1,243.18 | 13 | 14 | 15 | 2,000 persisted insufficient outcomes |

Replay is read-only after acquiring the campaign serialization lock. An
insufficient command writes an immutable completed receipt, explaining its
higher cost.

## Phase 2 decision-read impact

The decision fixture has 1,000 eligible campaigns. Each mixed run used 16
decision VUs plus 16 accounting VUs.

| Workload | Decision requests | Decision req/s | Decision p50/p95/p99 ms | Accounting requests | Accounting req/s | Accounting p50/p95/p99 ms |
| --- | ---: | ---: | --- | ---: | ---: | --- |
| Decisions only | 21,108 | 1,406.19 | 11 / 14 / 16 | — | — | — |
| Decisions + hot writes | 12,607 | 839.33 | 19 / 21 / 24 | 13,292 | 884.94 | 18 / 20 / 23 |
| Decisions + distributed writes | 13,323 | 887.22 | 18 / 20 / 22 | 14,261 | 949.69 | 17 / 19 / 21 |

Accounting writes materially reduce decision throughput and increase decision
latency on this four-connection deployment. Hot-row serialization is visible,
but shared pool and PostgreSQL pressure also affect the distributed workload.
This is evidence to revisit capacity and serving isolation in later phases, not
justification to change the approved Phase 3 authority before its baseline.

## Final financial state

After all accounting fixtures, including additional timing runs:

- discrepancy query rows: `0`;
- PostgreSQL sessions during verification: `5`;
- hot campaign: configured `1,000,000,000`, committed `35,292`, remaining
  `999,964,708`;
- replay campaign: configured `1,000,000,000`, committed `1`, remaining
  `999,999,999`;
- insufficient campaign: configured `1`, committed `1`, remaining `0`.

The final committed values include exploratory reruns. They are reported only
as ledger/counter verification evidence, not as per-scenario throughput data.

## Reproduction

```sh
MERCURY_PERFORMANCE_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury_performance_test?sslmode=disable' \
  make performance-seed

MERCURY_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury_performance_test?sslmode=disable' \
MERCURY_HTTP_ADDRESS='127.0.0.1:18080' \
  go run ./cmd/mercury

docker run --rm --network host --user 1000:1000 \
  -v "$PWD:/workspace" -w /workspace \
  -e BASE_URL=http://127.0.0.1:18080 -e MODE=hot -e VUS=16 \
  -e ITERATIONS=2000 -e RUN_ID=hot-c16 \
  -e SUMMARY_PATH=loadtests/results/phase3-hot-c16.json \
  grafana/k6:1.5.0 run --quiet loadtests/phase3-budget-accounting.js
```

Supported modes are `hot`, `spread`, `replay`, `insufficient`,
`decision_only`, `mixed_hot`, and `mixed_spread`. Recreate the performance
database before a complete reproduction so idempotency keys and balances begin
from the documented fixture.
