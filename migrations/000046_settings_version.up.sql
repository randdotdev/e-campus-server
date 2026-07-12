-- Optimistic-concurrency token for the single-row settings table.
-- UpdatePartial reads (settings, version), then writes WHERE version = $expected,
-- so concurrent admin edits can no longer silently clobber one another.
ALTER TABLE settings ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
