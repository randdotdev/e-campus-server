-- Optimistic-concurrency token (Shape 1) for the institution's subscription
-- row. Tier and override edits re-read (row, version) and write WHERE version =
-- $expected, so concurrent platform-admin changes merge on retry instead of one
-- silently clobbering the other. Lost-update protection holds across replicas.
ALTER TABLE subscription ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
