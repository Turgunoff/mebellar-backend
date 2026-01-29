-- Migration: Convert product name and description to JSONB for multi-language support
-- Date: 2024
-- Description: Convert product name and description from VARCHAR/TEXT to JSONB to support uz, ru, en translations

-- Step 1: Add new JSONB columns
ALTER TABLE products ADD COLUMN IF NOT EXISTS name_jsonb JSONB;
ALTER TABLE products ADD COLUMN IF NOT EXISTS description_jsonb JSONB;

-- Step 2: Migrate existing data
-- Each existing name/description will be set for all 3 languages (uz, ru, en)
UPDATE products
SET name_jsonb = jsonb_build_object(
  'uz', COALESCE(name, ''),
  'ru', COALESCE(name, ''),
  'en', COALESCE(name, '')
)
WHERE name_jsonb IS NULL AND name IS NOT NULL;

UPDATE products
SET description_jsonb = jsonb_build_object(
  'uz', COALESCE(description, ''),
  'ru', COALESCE(description, ''),
  'en', COALESCE(description, '')
)
WHERE description_jsonb IS NULL;

-- Step 3: Set default for new rows
UPDATE products
SET name_jsonb = jsonb_build_object('uz', '', 'ru', '', 'en', '')
WHERE name_jsonb IS NULL;

UPDATE products
SET description_jsonb = jsonb_build_object('uz', '', 'ru', '', 'en', '')
WHERE description_jsonb IS NULL;

-- Step 4: Drop old columns
ALTER TABLE products DROP COLUMN IF EXISTS name;
ALTER TABLE products DROP COLUMN IF EXISTS description;

-- Step 5: Rename new columns
ALTER TABLE products RENAME COLUMN name_jsonb TO name;
ALTER TABLE products RENAME COLUMN description_jsonb TO description;

-- Step 6: Add constraints to ensure name is not null and contains required keys
ALTER TABLE products
ADD CONSTRAINT products_name_required_keys 
CHECK (name IS NOT NULL AND name ? 'uz');

-- Step 7: Create indexes on name and description for JSONB queries (optional but recommended)
CREATE INDEX IF NOT EXISTS idx_products_name_jsonb ON products USING GIN (name);
CREATE INDEX IF NOT EXISTS idx_products_description_jsonb ON products USING GIN (description);
