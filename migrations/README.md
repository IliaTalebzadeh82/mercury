# Production migrations

This directory is the only production migration source. Phase 1 introduces the
campaign control-plane schema in `00001_phase_one_control_plane.sql`. Phase 2
adds only the country-first decision lookup index in
`00002_ad_decision_country_lookup.sql`.

The production Goose up/down/reapply workflow runs against the dedicated
disposable `mercury_migration_test` database. The Phase 0-only migration fixture
remains under `testdata/migrations`; it is not part of this production set.
