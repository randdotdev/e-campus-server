-- Rename _ku to _local for consistency
ALTER TABLE users RENAME COLUMN full_name_ku TO full_name_local;
ALTER TABLE colleges RENAME COLUMN name_ku TO name_local;
ALTER TABLE departments RENAME COLUMN name_ku TO name_local;
ALTER TABLE programs RENAME COLUMN name_ku TO name_local;

-- Make courses bilingual
ALTER TABLE courses RENAME COLUMN name TO name_en;
ALTER TABLE courses ADD COLUMN name_local VARCHAR(255);
ALTER TABLE courses RENAME COLUMN subtitle TO subtitle_en;
ALTER TABLE courses ADD COLUMN subtitle_local VARCHAR(100);
ALTER TABLE courses RENAME COLUMN description TO description_en;
ALTER TABLE courses ADD COLUMN description_local TEXT;

-- Make roles bilingual
ALTER TABLE roles RENAME COLUMN title TO title_en;
ALTER TABLE roles ADD COLUMN title_local VARCHAR(100);

-- Add user preferences
ALTER TABLE users ADD COLUMN preferred_language VARCHAR(5) NOT NULL DEFAULT 'en';
ALTER TABLE users ADD COLUMN timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Baghdad';
ALTER TABLE users ADD COLUMN theme VARCHAR(10) NOT NULL DEFAULT 'system';
