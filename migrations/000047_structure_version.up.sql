-- Optimistic-concurrency tokens for the university-structure tables.
-- Each Update re-reads (row, version) and writes WHERE version = $expected, so
-- concurrent admin edits merge on retry instead of silently clobbering one
-- another. Lost-update protection holds across application replicas.
ALTER TABLE colleges ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE departments ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE programs ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
