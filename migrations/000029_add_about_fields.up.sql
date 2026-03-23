-- Add localized about fields to colleges (JSONB for flexibility)
ALTER TABLE colleges
    ADD COLUMN about JSONB DEFAULT '{}',
    ADD COLUMN founded INT,
    ADD COLUMN phone VARCHAR(50),
    ADD COLUMN email VARCHAR(255),
    ADD COLUMN logo_url TEXT;

-- Migrate existing description to about.en
UPDATE colleges SET about = jsonb_build_object('en', description) WHERE description IS NOT NULL;

-- Add localized about fields to departments
ALTER TABLE departments
    ADD COLUMN about JSONB DEFAULT '{}',
    ADD COLUMN founded INT,
    ADD COLUMN phone VARCHAR(50),
    ADD COLUMN email VARCHAR(255),
    ADD COLUMN logo_url TEXT;

-- Migrate existing description to about.en
UPDATE departments SET about = jsonb_build_object('en', description) WHERE description IS NOT NULL;
