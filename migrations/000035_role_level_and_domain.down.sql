ALTER TABLE roles DROP COLUMN domain;
ALTER TABLE roles RENAME COLUMN level TO permission;
