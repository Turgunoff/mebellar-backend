-- Migration: Create banners table with JSONB localization support
-- Date: 2026-01-25

-- Create banners table
CREATE TABLE IF NOT EXISTS banners (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title JSONB NOT NULL,           -- Stores: {"uz": "...", "ru": "...", "en": "..."}
    subtitle JSONB,                 -- Stores: {"uz": "...", "ru": "...", "en": "..."}
    image_url TEXT NOT NULL,
    target_type VARCHAR(50) DEFAULT 'none',  -- none, category, product, external
    CONSTRAINT check_target_type CHECK (target_type IN ('none', 'category', 'product', 'external'))
    target_id VARCHAR(255),         -- ID of category/product or external URL
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_banners_is_active ON banners(is_active);
CREATE INDEX IF NOT EXISTS idx_banners_sort_order ON banners(sort_order);

-- Insert sample banners for testing
INSERT INTO banners (title, subtitle, image_url, target_type, sort_order, is_active) VALUES
(
    '{"uz": "Yangi Kolleksiya", "ru": "Новая Коллекция", "en": "New Collection"}',
    '{"uz": "30% gacha chegirma", "ru": "Скидки до 30%", "en": "Up to 30% off"}',
    'https://images.unsplash.com/photo-1555041469-a586c61ea9bc?w=800',
    'none',
    0,
    true
),
(
    '{"uz": "Premium Divanlar", "ru": "Премиум Диваны", "en": "Premium Sofas"}',
    '{"uz": "Maxsus narxlarda", "ru": "По специальным ценам", "en": "Special prices"}',
    'https://images.unsplash.com/photo-1493663284031-b7e3aefcae8e?w=800',
    'category',
    1,
    true
),
(
    '{"uz": "Yotoqxona to''plami", "ru": "Спальный комплект", "en": "Bedroom Set"}',
    '{"uz": "Bepul yetkazib berish", "ru": "Бесплатная доставка", "en": "Free delivery"}',
    'https://images.unsplash.com/photo-1538688525198-9b88f6f53126?w=800',
    'none',
    2,
    true
);
