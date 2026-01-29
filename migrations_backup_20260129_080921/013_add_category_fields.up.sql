-- Add slug, is_active, and sort_order to categories table
ALTER TABLE categories 
ADD COLUMN IF NOT EXISTS slug VARCHAR(255) UNIQUE,
ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT TRUE,
ADD COLUMN IF NOT EXISTS sort_order INTEGER DEFAULT 0;

-- Create index on slug for faster lookups
CREATE INDEX IF NOT EXISTS idx_categories_slug ON categories(slug);

-- Create index on is_active for filtering
CREATE INDEX IF NOT EXISTS idx_categories_is_active ON categories(is_active);

-- Create index on sort_order for ordering
CREATE INDEX IF NOT EXISTS idx_categories_sort_order ON categories(sort_order);

-- Generate slugs for existing categories (if any)
-- This will be handled by the backend when updating categories
