ALTER TABLE colleges ADD COLUMN description_temp JSONB DEFAULT '{}';
UPDATE colleges SET description_temp = jsonb_build_object('en', description) WHERE description IS NOT NULL;
ALTER TABLE colleges DROP COLUMN description;
ALTER TABLE colleges RENAME COLUMN description_temp TO description;

ALTER TABLE departments ADD COLUMN description_temp JSONB DEFAULT '{}';
UPDATE departments SET description_temp = jsonb_build_object('en', description) WHERE description IS NOT NULL;
ALTER TABLE departments DROP COLUMN description;
ALTER TABLE departments RENAME COLUMN description_temp TO description;
