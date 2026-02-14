ALTER TABLE programs
ADD COLUMN min_age INT CHECK (min_age IS NULL OR min_age >= 0),
ADD COLUMN max_age INT CHECK (max_age IS NULL OR max_age <= 100),
ADD CONSTRAINT programs_age_range_valid CHECK (min_age IS NULL OR max_age IS NULL OR min_age <= max_age);
