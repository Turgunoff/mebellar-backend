-- ============================================
-- MIGRATION: Create shops table
-- Version: 016
-- Description: Do'konlar jadvali (Multi-language support with JSONB)
-- ============================================

-- ============================================
-- UP MIGRATION
-- ============================================

-- Shops jadvali (One Seller -> Many Shops)
CREATE TABLE IF NOT EXISTS shops (
    -- Asosiy identifikatorlar
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES seller_profiles(id) ON DELETE CASCADE,
    
    -- Multi-language maydonlar (JSONB)
    name JSONB NOT NULL DEFAULT '{}', -- {"uz": "...", "ru": "...", "en": "..."}
    description JSONB DEFAULT '{}',    -- {"uz": "...", "ru": "...", "en": "..."}
    address JSONB DEFAULT '{}',        -- {"uz": "...", "ru": "...", "en": "..."}
    
    -- SEO va identifikatsiya
    slug VARCHAR(255) UNIQUE,
    
    -- Media
    logo_url VARCHAR(500),
    banner_url VARCHAR(500),
    
    -- Aloqa va joylashuv
    phone VARCHAR(20),
    latitude FLOAT8,
    longitude FLOAT8,
    region_id UUID REFERENCES regions(id) ON DELETE SET NULL,
    
    -- Ish vaqtlari (JSONB)
    working_hours JSONB DEFAULT '{}',
    
    -- Status va reyting
    is_active BOOLEAN DEFAULT TRUE,
    is_verified BOOLEAN DEFAULT FALSE,
    is_main BOOLEAN DEFAULT FALSE, -- Asosiy do'kon (bir seller uchun faqat bitta)
    rating FLOAT DEFAULT 0,
    
    -- Vaqt belgilari
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indekslar
CREATE INDEX IF NOT EXISTS idx_shops_seller_id ON shops(seller_id);
CREATE INDEX IF NOT EXISTS idx_shops_slug ON shops(slug);
CREATE INDEX IF NOT EXISTS idx_shops_region_id ON shops(region_id);
CREATE INDEX IF NOT EXISTS idx_shops_is_active ON shops(is_active);
CREATE INDEX IF NOT EXISTS idx_shops_is_verified ON shops(is_verified);
CREATE INDEX IF NOT EXISTS idx_shops_is_main ON shops(is_main);

-- GIN indekslar JSONB maydonlar uchun
CREATE INDEX IF NOT EXISTS idx_shops_name_gin ON shops USING GIN (name);
CREATE INDEX IF NOT EXISTS idx_shops_address_gin ON shops USING GIN (address);

-- Trigger: updated_at avtomatik yangilash
CREATE OR REPLACE FUNCTION update_shops_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_shops_updated_at ON shops;
CREATE TRIGGER trigger_shops_updated_at
    BEFORE UPDATE ON shops
    FOR EACH ROW
    EXECUTE FUNCTION update_shops_updated_at();

-- Constraint: Bir seller uchun faqat bitta asosiy do'kon bo'lishi kerak
CREATE UNIQUE INDEX IF NOT EXISTS idx_shops_seller_main 
ON shops(seller_id) 
WHERE is_main = TRUE;

-- Izohlar
COMMENT ON TABLE shops IS 'Sotuvchi do''konlari (bir seller ko''p do''kon ochishi mumkin)';
COMMENT ON COLUMN shops.seller_id IS 'Bog''langan seller profile ID';
COMMENT ON COLUMN shops.name IS 'Do''kon nomi (multi-language JSONB)';
COMMENT ON COLUMN shops.description IS 'Do''kon tavsifi (multi-language JSONB)';
COMMENT ON COLUMN shops.address IS 'Do''kon manzili (multi-language JSONB)';
COMMENT ON COLUMN shops.slug IS 'SEO-friendly URL (e.g., /shop/mebel-house-tashkent)';
COMMENT ON COLUMN shops.working_hours IS 'Ish vaqtlari (JSON)';
COMMENT ON COLUMN shops.is_main IS 'Asosiy do''kon (bir seller uchun faqat bitta)';
COMMENT ON COLUMN shops.is_verified IS 'Admin tomonidan tasdiqlangan';

-- ============================================
-- DOWN MIGRATION (Rollback)
-- ============================================
-- Agar tiklash kerak bo'lsa quyidagini ishlatish:
/*
DROP TRIGGER IF EXISTS trigger_shops_updated_at ON shops;
DROP FUNCTION IF EXISTS update_shops_updated_at();
DROP INDEX IF EXISTS idx_shops_name_gin;
DROP INDEX IF EXISTS idx_shops_address_gin;
DROP INDEX IF EXISTS idx_shops_seller_main;
DROP TABLE IF EXISTS shops;
*/
