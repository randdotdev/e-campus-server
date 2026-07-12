-- Optimistic-concurrency tokens (Shape 1) for the announcements aggregates.
-- Post and activity edits re-read (row, version) and write WHERE version =
-- $expected, so a concurrent body edit and pin toggle merge on retry instead
-- of one silently clobbering the other. Lost-update protection holds across
-- application replicas.
ALTER TABLE posts ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
ALTER TABLE activities ADD COLUMN version BIGINT NOT NULL DEFAULT 0;
