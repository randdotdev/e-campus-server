-- Remove user preferences
ALTER TABLE users DROP COLUMN theme;
ALTER TABLE users DROP COLUMN timezone;
ALTER TABLE users DROP COLUMN preferred_language;

-- Revert roles
ALTER TABLE roles DROP COLUMN title_local;
ALTER TABLE roles RENAME COLUMN title_en TO title;

-- Revert courses
ALTER TABLE courses DROP COLUMN description_local;
ALTER TABLE courses RENAME COLUMN description_en TO description;
ALTER TABLE courses DROP COLUMN subtitle_local;
ALTER TABLE courses RENAME COLUMN subtitle_en TO subtitle;
ALTER TABLE courses DROP COLUMN name_local;
ALTER TABLE courses RENAME COLUMN name_en TO name;

-- Revert _local to _ku
ALTER TABLE programs RENAME COLUMN name_local TO name_ku;
ALTER TABLE departments RENAME COLUMN name_local TO name_ku;
ALTER TABLE colleges RENAME COLUMN name_local TO name_ku;
ALTER TABLE users RENAME COLUMN full_name_local TO full_name_ku;
