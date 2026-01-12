-- +migrate Up
-- Add shop_id column to products table for multi-shop support

ALTER TABLE products 
ADD COLUMN IF NOT EXISTS shop_id UUID REFERENCES seller_profiles(id) ON DELETE SET NULL;

-- Create index for better query performance
CREATE INDEX IF NOT EXISTS idx_products_shop_id ON products(shop_id);

-- +migrate Down
DROP INDEX IF EXISTS idx_products_shop_id;
ALTER TABLE products DROP COLUMN IF EXISTS shop_id;
