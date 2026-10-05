# Mercury code map

Concise ownership map for targeted inspection. Use search and read only the relevant entry points/tests.

| Area | Ownership and entry points | Important tests/evidence | Key invariant/boundary |
| --- | --- | --- | --- |
| `cmd/mercury` | Composition root; wires configuration, database, domains, API, and server. | Broad build/integration through package tests. | Composition, not domain ownership. |
| `internal/platform/config` | Environment parsing/validation. | `config_test.go` | Invalid required configuration fails before serving. |
| `internal/platform/database` | `pgxpool` creation/connectivity lifecycle. | database unit/integration tests | PostgreSQL is current authority; pool is bounded. |
| `internal/platform/server` | HTTP lifecycle, health/readiness, shutdown. | `server_test.go` | Liveness is process health; current readiness includes database reachability. |
| `internal/api` | HTTP adapters and route composition for advertisers, campaigns, decisions, budgets. | `api_integration_test.go` | Translates protocols; does not own domain rules. |
| `internal/advertiser` | Advertiser creation and lookup. | unit/integration tests | Existing advertiser required by campaigns. |
| `internal/campaign` | Campaign configuration, lifecycle, targeting, versions, PostgreSQL transactions. | campaign unit/integration tests | See Phase 1 section of `docs/invariants.md`. |
| `internal/decision` | Opportunity normalization, single-statement selection, ranking, transient identity, diagnostics. | decision unit/integration tests; Phase 2 report | Exact eligibility/ranking; no-fill is not database failure. |
| `internal/budget` | Budget snapshots, immediate consumption, idempotency, immutable receipts. | budget unit/integration tests; Phase 3 report | Exact minor units and campaign-row serialization. |
| `migrations` | Production schema and migration history. | `testdata/migrations/verify.sh` | Dedicated migration database; preserve up/down/reapply behavior. |
| `loadtests` / `testdata/performance` | Reproducible decision and budget workloads/fixtures. | `docs/performance/` | Report environment/dataset/concurrency and do not overclaim. |
| `web` | Next.js product/demo UI, server actions/API adapters, system health, campaign and Decision Lab views. | colocated Vitest tests and `web/e2e` | Backend is authoritative; backend URL stays server-side. |
| `adr` | Historical architectural decisions 001–009. | ADR status/consequences | Do not rewrite history; add ADRs for new durable choices. |
| `docs/performance` | Retained measured Phase 2–3 baselines. | linked fixtures/results | Historical evidence, not default startup context. |

