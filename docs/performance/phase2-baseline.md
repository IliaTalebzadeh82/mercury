# Phase 2 exploratory decision baseline

Measured 2026-09-25. These results characterize one development machine and
are not an SLO or a production-capacity claim.

## Environment and method

- Host: Intel Core i7-10750H, 6 cores/12 threads, 15 GiB RAM, Linux x86-64.
- PostgreSQL: 17.6 Alpine in Docker Compose, with no configured CPU or memory limit.
- Backend: host process, no configured CPU or memory limit, Go pool fixed at 4 connections.
- Driver: `grafana/k6:1.5.0` using constant arrival rate.
- Every run: 30-second warmup followed by a 120-second measured window.
- Dataset: 1,211 campaigns—exact eligible cohorts of 0/1/10/100/1000 plus
  100 rejected background campaigns (25 DRAFT, 25 PAUSED, 25 wrong placement,
  and 25 wrong country). Placement is `search_results`.
- Countries `XA` through `XF` are reproducible synthetic codes. Phase 2 checks
  two-letter uppercase shape; it does not claim an authoritative ISO catalog.
- Achieved RPS is measured-period request count divided by 120 seconds. k6's
  summary `rate` spans scenario start offsets, so it is not used here.
- Error rate is the k6 HTTP failure rate. All runs had zero failed response-shape
  checks and zero dropped iterations.

## Candidate-count comparison at 50 requested RPS

| Eligible | Measured requests | Achieved RPS | p50 ms | p95 ms | p99 ms | Error | No-fill |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 0 | 6,000 | 50.000 | 0.993 | 3.760 | 4.254 | 0% | 100% |
| 1 | 6,001 | 50.008 | 1.380 | 4.021 | 4.825 | 0% | 0% |
| 10 | 6,001 | 50.008 | 1.719 | 4.926 | 6.065 | 0% | 0% |
| 100 | 6,001 | 50.008 | 1.442 | 5.823 | 6.851 | 0% | 0% |
| 1000 | 6,001 | 50.008 | 2.803 | 4.211 | 13.387 | 0% | 0% |

The single-host results are not perfectly monotonic at every percentile, but
the 1000-candidate cohort has the clearest higher median and tail. The query
plans below show the underlying ranking work more directly.

## Mixed-cohort rate exploration

The mixed scenario cycles evenly through all five cohorts, hence the expected
20% no-fill rate.

| Requested RPS | Measured requests | Achieved RPS | p50 ms | p95 ms | p99 ms | Error | No-fill |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 10 | 1,200 | 10.000 | 1.942 | 5.432 | 9.135 | 0% | 20.000% |
| 50 | 6,001 | 50.008 | 1.260 | 3.355 | 5.696 | 0% | 20.013% |
| 100 | 12,001 | 100.008 | 1.142 | 2.832 | 3.228 | 0% | 20.007% |
| 200 | 24,000 | 200.000 | 0.837 | 2.542 | 2.822 | 0% | 20.000% |

This machine sustained the explored 200 RPS input without HTTP failures or
dropped iterations. That is evidence only for this fixture and environment,
not a production throughput guarantee. Lower latency at higher arrival rates is
consistent with a warmed local system and measurement noise; it is not evidence
that load improves latency.

During a separate identical 200 RPS resource run (24,001 requests, p50 0.727
ms, p95 2.509 ms, p99 2.748 ms, zero errors/drops), five one-second `pidstat`
samples reported backend CPU averaging 4.8%; RSS was 29,968 KiB. One
`docker stats` snapshot reported PostgreSQL at 24.5% CPU and 68.07 MiB. The
database showed five sessions: the four-connection application pool plus the
measurement session. These point-in-time values are diagnostic snapshots, not
time-series utilization measurements.

## Query-plan evidence

`EXPLAIN (ANALYZE, BUFFERS)` was executed from
`testdata/performance/explain.sql` against every cohort after seeding:

| Eligible | Execution ms | Shared hits | Country access |
| ---: | ---: | ---: | --- |
| 0 | 0.141 | 10 | country/campaign index-only scan |
| 1 | 0.248 | 8 | country/campaign index-only scan |
| 10 | 0.367 | 35 | country/campaign index-only scan + campaign hash join |
| 100 | 0.501 | 35 | country/campaign index-only scan + campaign hash join |
| 1000 | 2.065 | 40 | sequential target scan + campaign hash join |

At 1000 eligible campaigns, 1,075 target rows have country `XE` because the
background DRAFT/PAUSED/wrong-placement campaigns deliberately share it; the
state and placement filters yield exactly 1000 eligible rows. PostgreSQL's
sequential scan at that selectivity is reasonable and is not overridden.

## Reproduction

```sh
MERCURY_PERFORMANCE_DATABASE_URL='postgres://mercury:mercury@localhost:55432/mercury_performance_test?sslmode=disable' \
  make performance-seed

docker compose exec -T postgres psql -X -v ON_ERROR_STOP=1 \
  -U mercury -d mercury_performance_test -f /dev/stdin \
  < testdata/performance/explain.sql

docker run --rm --network host --user 1000:1000 \
  -v "$PWD:/workspace" -w /workspace \
  -e BASE_URL=http://127.0.0.1:18080 -e COHORT=mixed -e RATE=200 \
  -e SUMMARY_PATH=loadtests/results/mixed-200.json \
  grafana/k6:1.5.0 run --quiet loadtests/phase2-ad-decisions.js
```
