-- ============================================
-- MIGRATION: Create seller_profiles table
-- Version: 002
-- Description: Sotuvchi profillari jadvali
-- ============================================

-- ============================================
-- UP MIGRATION
-- ============================================

-- Seller Profiles jadvali (Multi-Shop: One User -> Many Shops)
CREATE TABLE IF NOT EXISTS seller_profiles (
    -- Asosiy identifikatorlar
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE, -- NOT UNIQUE: allows multiple shops per user
    
    -- Biznes ma'lumotlari
    shop_name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE,
    description TEXT,
    logo_url VARCHAR(500),
    banner_url VARCHAR(500),
    
    -- Yuridik va moliyaviy ma'lumotlar (maxfiy)
    legal_name VARCHAR(255),
    tax_id VARCHAR(50),
    bank_account VARCHAR(50),
    bank_name VARCHAR(255),
    
    -- Aloqa va joylashuv
    support_phone VARCHAR(20),
    address VARCHAR(500),
    latitude FLOAT8,
    longitude FLOAT8,
    
    -- JSONB maydonlari
    social_links JSONB DEFAULT '{}',
    working_hours JSONB DEFAULT '{}',
    
    -- Status va reyting
    is_verified BOOLEAN DEFAULT FALSE,
    rating FLOAT DEFAULT 0,
    
    -- Vaqt belgilari
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Indekslar
CREATE INDEX IF NOT EXISTS idx_seller_profiles_user_id ON seller_profiles(user_id);
CREATE INDEX IF NOT EXISTS idx_seller_profiles_shop_name ON seller_profiles(shop_name);
CREATE INDEX IF NOT EXISTS idx_seller_profiles_slug ON seller_profiles(slug);
CREATE INDEX IF NOT EXISTS idx_seller_profiles_is_verified ON seller_profiles(is_verified);

-- Trigger: updated_at avtomatik yangilash
CREATE OR REPLACE FUNCTION update_seller_profiles_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_seller_profiles_updated_at ON seller_profiles;
CREATE TRIGGER trigger_seller_profiles_updated_at
    BEFORE UPDATE ON seller_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_seller_profiles_updated_at();

-- Izoh qo'shish
COMMENT ON TABLE seller_profiles IS 'Sotuvchi do''kon profillari';
COMMENT ON COLUMN seller_profiles.user_id IS 'Bog''langan foydalanuvchi (One-to-One)';
COMMENT ON COLUMN seller_profiles.slug IS 'SEO-friendly URL (e.g., /shop/mebel-house)';
COMMENT ON COLUMN seller_profiles.social_links IS 'Ijtimoiy tarmoq havolalari (JSON)';
COMMENT ON COLUMN seller_profiles.working_hours IS 'Ish vaqtlari (JSON)';
COMMENT ON COLUMN seller_profiles.is_verified IS 'Admin tomonidan tasdiqlangan';
COMMENT ON COLUMN seller_profiles.rating IS 'O''rtacha reyting (cached)';

-- ============================================
-- DOWN MIGRATION (Rollback)
-- ============================================
-- Agar tiklash kerak bo'lsa quyidagini ishlatish:
/*
DROP TRIGGER IF EXISTS trigger_seller_profiles_updated_at ON seller_profiles;
DROP FUNCTION IF EXISTS update_seller_profiles_updated_at();
DROP TABLE IF EXISTS seller_profiles;
*/
