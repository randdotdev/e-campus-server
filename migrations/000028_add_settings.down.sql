DROP TABLE IF EXISTS user_preferences;

-- Revert settings to simple structure
UPDATE settings SET settings = jsonb_build_object(
    'full_year_repeat', COALESCE((settings->'features'->>'full_year_repeat')::boolean, false)
);
