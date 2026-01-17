-- Convert category name from VARCHAR to JSONB for multi-language support
-- This migration safely converts existing data by creating JSON objects with all 3 languages

-- Step 1: Add a temporary column for the JSONB data
ALTER TABLE categories ADD COLUMN IF NOT EXISTS name_jsonb JSONB;

-- Step 2: Convert existing VARCHAR data to JSONB format
-- Each existing name will be set for all 3 languages (uz, ru, en)
UPDATE categories 
SET name_jsonb = jsonb_build_object(
  'uz', COALESCE(name, ''),
  'ru', COALESCE(name, ''),
  'en', COALESCE(name, '')
)
WHERE name_jsonb IS NULL;

-- Step 3: Drop the old VARCHAR column
ALTER TABLE categories DROP COLUMN IF EXISTS name;

-- Step 4: Rename the new column to 'name'
ALTER TABLE categories RENAME COLUMN name_jsonb TO name;

-- Step 5: Add constraint to ensure name is not null and contains required keys
ALTER TABLE categories 
ADD CONSTRAINT categories_name_required_keys 
CHECK (name IS NOT NULL AND name ? 'uz' AND name ? 'ru' AND name ? 'en');

-- Step 6: Create index on name for JSONB queries (optional but recommended)
CREATE INDEX IF NOT EXISTS idx_categories_name_jsonb ON categories USING GIN (name);
