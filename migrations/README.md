# Production migrations

This directory intentionally contains no SQL migration in Phase 0. Mercury has
no production-owned domain schema yet, and the project does not create a fake
table merely to exercise migration tooling.

The Goose up/down/reapply workflow is verified with the reversible fixture in
`testdata/migrations` against a dedicated disposable migration-test database.
Production migrations will be added here alongside the domain state they
introduce.
