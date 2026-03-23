-- Remove about/contact fields from colleges
ALTER TABLE colleges
    DROP COLUMN IF EXISTS about,
    DROP COLUMN IF EXISTS founded,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS logo_url;

-- Remove about/contact fields from departments
ALTER TABLE departments
    DROP COLUMN IF EXISTS about,
    DROP COLUMN IF EXISTS founded,
    DROP COLUMN IF EXISTS phone,
    DROP COLUMN IF EXISTS email,
    DROP COLUMN IF EXISTS logo_url;
