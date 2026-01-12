-- Migration: Add analytics columns to products table
-- Date: 2026-01-12

-- Add sold_count column (buyurtma orqali sotilgan mahsulotlar soni)
ALTER TABLE products ADD COLUMN IF NOT EXISTS sold_count INT DEFAULT 0;

-- Add view_count column (ko'rilganlar soni)
ALTER TABLE products ADD COLUMN IF NOT EXISTS view_count INT DEFAULT 0;

-- Create indexes for sorting performance
CREATE INDEX IF NOT EXISTS idx_products_sold_count ON products(sold_count DESC);
CREATE INDEX IF NOT EXISTS idx_products_view_count ON products(view_count DESC);
CREATE INDEX IF NOT EXISTS idx_products_rating ON products(rating DESC);
CREATE INDEX IF NOT EXISTS idx_products_price ON products(price ASC);

-- Verify columns
-- SELECT column_name, data_type, column_default 
-- FROM information_schema.columns 
-- WHERE table_name = 'products' AND column_name IN ('sold_count', 'view_count');
